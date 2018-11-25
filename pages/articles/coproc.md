# 并发处理模型, 从 Reactor 到 Coproc

*这是2010年左右的老文，对于分析理清各种流行并发模型，它还是有帮助的*

## 简介

本文介绍一个正在开发的 C++ 服务框架 coproc 在并发处理模型上的设计思路. coproc 基于 libevent 和基本的 Reactor 模型, 在此之上逐步实现了轻进程以及类似 UNIX fork-wait 的并发模型, 并利用 ucontext coroutine 机制实现了真正的"进程"上下文切换. 从而实现从事件驱动到顺序处理, 从异步到同步等待的模型进化. 在实现 UNIX 传统并发抽象的同时, 保持了相当高的运行效率.

## Reactor

Reactor 是很常见的并发处理模型, 将业务组织成各种事件驱动的对象. 一个监听接收新 TCP 连接的 Reactor 会像是:

	class ListenReactor : public Reactor {
	  ...
	  void OnSockEvent(int listen_fd, int ev) {
        int sock = accept(listen_fd, NULL, 0);
        ...
  	  }
    }

利用 libevent 作为事件驱动器, 经过少许包装, 让 Reactor 可以注册到句柄事件上, 回调 OnSockEvent, 这段代码就可以运作了, 非常简单.

NOTE: 在实现中, 我们有任务/消息队列实现异步的多类型消息/事件分发, 下面的事件触发或消息传送或方法调用 (都是一样的东西) 在模型上都是异步的, 同步直接调用只是一个 shortcut, 下面不再重复. 消息队列是支持跨线程协同, 容量控制等的关键, 也是性能优化的重点, 在后面, 它还担负类似 OS 任务调度的工作 (所以它叫 Scheduler), 但与这里讨论的问题无关, 在此略过.

## 模型改进

"监听接收新连接" 这个任务从文字上就是事件驱动的. 但 "从 socket 读取 N 个字节" 这个任务不是, 它本来就是过程式的. Web 应用服务所要处理的绝大多数业务属于后者, 从模型上并不与事件驱动模型吻合. 将一个有复杂业务流程和 IO 交互方式的 Web 应用完全组织成事件驱动的 Reactor 方式, 无论设计, 编程, 调试还是后续的维护都是一个极大的挑战.

我们需要在 Reactor 模型之上实现异步的 调用-返回语义, 这是过程式处理逻辑中最常见的模式:

一个任务由一个独立 Reactor 处理. 例如不存在管理程序全部 socket, 提供统一 IO 服务的 Reactor. 而是一个 IO 交互任务, 则创建一个专用的 Reactor 来完成.
引入标准事件: OnInit(), 在 Reactor 任务开始时触发, OnStop(), 在 Reactor 因任何原因停止 (自行停止或他杀) 时触发.
引入关联 link 机制: 若 A 与 B 关联, 则当 B 停止时, 触发 A 的 OnLinkReturn(B) 事件.
由此, 我们在 Reactor 上实现了与过程式处理中调用-返回对应的模型:

调用: 创建子任务 Reactor, 将其与父任务关联, 并启动之. 我们把这个过程称做 Spawn
返回: 子任务完成时触发父任务 OnLinkReturn 事件, 在事件处理中, 父任务可获取子任务完成状态等信息, 然后销毁子任务 Reactor, 完成这个 Reactor 生命周期.
在这种模型下, 我们可以用熟悉的思路实现对逻辑的分层封装. 在良好的封装设计下, 除了最底层的 IO 处理 Reactor 外, 其他的 Reactor 只需处理 OnLinkReturn 事件, 根据各种子任务的返回改变状态, 驱动下一步处理, 即可顺利完成任务.

一个 RPC Client 的处理可以实现为 (随手写的, 不用深究细节):

    class RPCClient : public Reactor {
      int m_stage;

      virtual void OnInit() {
        m_stage = STAGE_SEND;
        m_send_reactor.SetRequest(request);
        Spawn(&m_send_reactor);
      }

      virtual void OnLinkReturn(Reactor *src) {
        if (m_stage == STAGE_SEND) {
          m_stage = STAGE_RECV;
          Spawn(&m_recv_reactor);
        } else {
          Return(m_recv_reactor.GetResult());
        }
      }
    }

## Proc

在上例中, 我们的需求实际是等待发送完成, 再启动读取任务, 等待读取完成后再取出结果. 等待是这个任务的天然属性, 我们需要实现支持. 当然, 我们不会让等待任务阻塞在 OS 进程或线程上. 我们做的, 实际上只是对上面代码的一点小改进.

    class RPCClient : public Reactor {
      int m_next_stage;

      void OnInit() {
        Process(STAGE_INIT);
      }

      void Process(int stage) {
        switch (stage) {
        case STAGE_INIT:
          m_send_reactor.SetRequest(request);
          Spawn(&m_send_reactor);
          return SetWait(STAGE_SENT);
        case STAGE_SENT:
          Spawn(&m_recv_reactor);
          return SetWait(STAGE_RECV);
        case STAGE_RECV:
          return Return(m_recv_reactor.GetResult());
        }
      }

      void SetWait(int stage) {
        m_next_stage = stage;
      }

      void OnLinkReturn(Reactor *src) {
        Process(m_next_stage);
      }
    }

在这里, 我们把业务处理分成若干个阶段 (stage), 各阶段处理都是非阻塞的, 当它 Spawn 子任务, 并需要等待子任务返回以进入下一阶段时, 它设置等待被唤醒后的状态即可退出. 当它所等待的唯一事件 — 子 Reactor 返回到来时, OnLinkReturn 执行统一的任务: 根据上次设置的返回状态, 调用 Process 进行下一 stage 处理, 如此一步步下去, 直到任务完成停止.

我们没有引入新功能, 只是对代码结构进行了简单优化. 但我们发现, 在这里, 唯一一个事件处理流程也被统一化, 被框架接管. 处理流程被组织成已经与原本过程式处理非常类似的代码. 在上述简单实现中我们仅实现了对一个子任务的等待, 稍加修改, 即可实现等待所有子任务返回的语义. 这些模式也许不够灵活, 但常用且易用. 虽然只是一些简单的代码改造, 但现在, 我们的模型已经根本不是事件处理模型了.

我把这种模式叫做 Spawn-Wait 模式, 或者叫 fork-wait 模式. 这是 UNIX 最简单, 最直接的实现并发的方式: fork 一堆子进程做事, 然后等它们都返回. 我们在 Reactor 引入了进程状态概念: READY, WAIT, DONE, 我们以非常轻量的方法实现了基本的进程语义, 不同的是我们现在能非常高效地创建成千上万的进程, 让它们并发处理, 我把它称作轻进程. 在实现上, 它只是一个很简单的 Reactor 派生类, 可以叫它 Proc.

我们把任务组织成了父子进程/Reactor 的树状关系, 这也让 C++ 下非常头痛的资源管理有了新的出路. 在目前实现中, Proc 带有一个计时器 (啊, 你可以把它理解成发 SIGALRM 的), 在 Proc 超时时会强行终止 Proc, 由于 Proc 维护了所有正在等待的子 Proc/Reactor 列表, Proc 终止时也会终止所有这些子进程, 有效回收所有资源 (内存, 句柄, 在 libevent 上的注册等等).

## Coproc

然则, 上面的处理模式仍然太笨拙. 说直白点, 这种依赖 stage 跳转的模式就是在用 GOTO 写程序, 更复杂的逻辑组织, 或是已有同步业务流程的移植依然是很困难的.

在 Proc 中, stage 的作用实际上是管理处理上下文, 当等待返回时, 通过 stage 回到恰当的处理逻辑中. 实际上, linux 本身就有 userland 的上下文切换管理机制 ucontext. 具体可参考 makecontext(3) 系列 manual.

使用这样的机制, 我们可以实现 Yield, Yield 可以在任何地方切换出当前处理流程, 进入 Proc 的等待状态, 而 OnLinkReturn 唤醒时, 则将 Yield 保存的上下文恢复, 继续 Yield 下面的流程. (NOTE: 这里的 Yield 比 python 的 yield 强, 它能在多层调用栈之上使用, 下文代码就是例子).

RPC Client 的例子太没难度, 下面逻辑实现两两并发调用 DB Reactor, 当它们返回结果总数超过 1000 时退出返回.
 
    class Searcher : public Coproc {
      const static int NUM_DBS = 256;

      int GetDouble(DB *db1, DB *db2) {
        Spawn(db1);
        Spawn(db2);
        Yield();
        return db1->GetNumResult() + db2->GetNumResult();
      }

      virtual int Main() {
        int total = 0;
        DB dbs[NUM_DBS];

        for (int i = 0; i < NUM_DBS; i += 2) {
          total += GetDouble(&dbs[i], &dbs[i + 1]);
          if (total >= 1000) {
            break;
          }
        }
        return total;
      }
    }

可以看到, 代码已经基本与同步逻辑代码无异了, 我们把这种模型称为 Coproc.

比之 Proc, Coproc 更进一步贴近了普通进程, 当然, 付出了预留 ucontext 栈空间和一点 ucontext 切换代价, 因而也更重了. 比之 OS 进程或线程, Coproc 不通用: 切换时机由自己控制, 不到自己 Yield 不会被换出, 一个长的纯 CPU 任务就能把系统堵死; 它不会自动处理会阻塞的 read(2) 等接口等等, 但这换来的就是小得多的资源消耗.

## 应用

虽然在 Reactor 上做了那么多事, 但令人吃惊的是, 对底层机制来说, 模型并没有什么改变: Reactor 依然处理事件, 只不过 Proc 和 Coproc 的事件处理被框架接管, 事件处理依然非阻塞. 对 libevent 来说, 整个系统没什么变化. 因此, 在底层, libevent 和 Reactor 模式依然用原来的方法保证系统良好的并发处理能力; 在上层, 从 Reactor 到 Coproc 都是互相兼容, 可以互相调用的. 开发者可以根据需求权衡, 选择最合适的模型.

例如, 在一个 Web 应用服务中. 端口的监听仍由基本 Reactor 实现, 而相对稳定通用的业务框架和数据访问层则由 Proc 实现 — 再复杂的 IO 模式都不在话下, 而业务逻辑则可以实现为 Coproc, 原同步框架下的代码也能很方便地移植过来. 一个请求到来时, 监听 Reactor 直接创建一个业务框架 Proc 来进行处理 — 这是 UNIX 最基本的 accept-fork 模型, 我喜欢这种优美自然的模式, 而且现在我可以尽情创建成千上万的 Proc. 业务框架 Proc 指挥整个业务处理流程, 涉及多个 Coproc 编写的业务逻辑模块, 虽然代码逻辑是同步的, 但 IO 能够天然并行.

## 总结

从 Reactor 到 Coproc, 我们不断进行权衡, 通过对模型加以恰当的限制和规范, 以及少许额外机制, 我们能从难以设计, 编程和调试的事件驱动模型进化到与普通同步模型相差无几. 在这个过程中, 我们很大程度上重造了轮子: OS, 我们在实现中不断加入了各种 OS 机制. 其实无论 IO 任务还是 OS 本身就是事件驱动的 — 由硬件中断驱动. 我们只是以稍微不同的, 严重轻量级的方法将事件驱动到顺序处理, 异步到同步阻塞的路重走了一遍而已.

我想, 这样做是有价值的, OS 本身对并行提供了优美, 成熟 — 更重要的是, 大家很熟悉的抽象. 但作为通用 OS, \*nix 不得不考虑更多情况, 实现更完整的封装和抽象, 导致其基本并发模型: 进程严重的 overhead. 以往, 我们要不对其并发模型作折衷: 例如, 放弃每个请求 fork 一个进程的做法, 而是使用进程或线程池, 更不会为并发请求而开一堆进程; 又或是另起炉灶, 完全放弃其模型, 例如采用全事件驱动模型. 而我的思路是, 在特定条件下, 我们可以保持这个成熟的模型, 而根据特定情况, 对 OS 机制的实现作权衡折衷.

这个思路显然受到 exokernel 研究和 erlang 的影响. 这里的轻进程模型显然没有 erlang 的灵活强大, 但相信对 UNIX 程序员来说, 会是一个更熟悉, 更易上手的模型.

本文提出的模型与 CERL 很类似, 但具体设计差异巨大. 本文的贡献在于, 展现参照 OS 设计, 从有成熟开源实现的 Reactor 模型到 Coproc 模型的思路, 说明模型选择并不是非此即彼, 而是可以根据情况权衡决定, 根据需求灵活定义 (除了 Proc 外, Reactor 还演化出了条件变量等不少模型) 的. CERL 项目远景宏大, 希望造出整套框架或系统, 而 coproc 专注于模型问题, 把 IO 事件处理 (libevent), 服务协议和描述 (protobuf etc) 等交给其他成熟的开源软件处理.
