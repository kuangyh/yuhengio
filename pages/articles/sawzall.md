@date Oct 02, 2016

# Sawzall, 及大数据的玩法


The Unofficial Google Data Science Blog 最近发了一篇 [关于 Dataflow 和 Spark 的文章](http://www.unofficialgoogledatascience.com/2016/08/next-generation-tools-for-data-science.html)，里面用一个 MH 估计（Mantel-Haenszel estimator）的例子来说明简单 MapReduce 计算模型的局限。MH 用来更准确地计算一个 A/B 实验的结果，防止复杂变动情况下的 bias. 比如文中研究实验对商品销售价格的影响，就先把销售流水按商品品类聚合，计算实验在每个品类上的结果，然后再做下一个聚合计算加权平均。因为有两层聚合，MR 实现至少要做成一个两级的 pipeline 写起来就头疼了。

可是这例子举得真心不好，虽然出自实际场景（还画蛇添足地说是 auction, 鬼都知道你在卖啥啦），但是高度抽象后问题整个就歪了：按文中的场景，假设你有 1000 万个 SKU, 不小了吧，不管你的销售流水记录有多大，一个简单的按 SKU 聚合的 MR 过后，结果只剩下 —— 每个商品两个实验，每个实验销售额和销量两个数据点，1000 万个 SKU = 4000 万个 int64. 一个可以轻松载入单机内存，用 R 或者 Python 随便把玩的数据量，搞那么复杂干嘛？是，文中描述的 MH 问题没法用单个 MR 解决，但是一个 MR 过后，它都已经不是一个需要分布式计算框架解决的问题了。

所以我最腻烦的就是那些张口闭口「大数据」，「分布式」的人，大你妹，分你妹。

在我看来，在所谓「大」数据的处理中，有两种截然不同的挑战。一种是传统意义的数据分析，从 MH 到更复杂的训练机器学习模型，难在计算模型的复杂多变；而另一种则是类似这里处理原始销售流水的问题，计算模型简单固定，难点是数据规模上的赤裸裸的大。这两个问题都很难而解决方法互相冲突，如果你觉得能用一个大一统方案同时解决两个问题，那八成是你遇到的两方面问题都不够难。像这里要实做实验分析平台处理全量 log，就应该是做个 daily MR 按预定的 dimension 组合做一层数据聚合解决「大」的问题，倒进 Dremel 或者 Mesa 之类的地方（数据不大时传统 RMDBS 也成）做数据仓库让人做复杂的在线查询分析，做模型也应该基于这份聚合好洗干净的数据，总之不会让你每次写个复杂的 pipeline 从原始流水搞起。

针对复杂计算模型问题，常见的解决思路就是像 Dataflow 和 Spark 一样，引入高层次抽象，一方面方便用户建模，另一方面用 planner 自动在分布式环境执行复杂数据流的计算，这方面的项目和经验已经很多了。但是后一种，字面意义上的「大」数据问题，就不是那么经常被提及了。

数据上了规模，你就无法承受任何不必要计算和数据传输的 overhead, 你必须把执行流程优化到无可再减；你没有也不应该有复杂的 DAG 所以也不需要计算框架的自动调度，你需要的是一个简单可控的架构，能轻易观察瓶颈所在，并做针对性优化。这真的不是一个 sexy 的问题，没有什么高大上的框架理论，就是看数据，优化 IO 优化算法，跑数据循环的苦逼活。Facebook 最近有一篇[关于 Spark 的文章](https://code.facebook.com/posts/1671373793181703/apache-spark-scale-a-60-tb-production-use-case/), 他们把一个 60+TB 的经典聚合+特征提取任务性能提升了好几倍，达成这个提高的并不是 Spark 的什么高级特性，只是将毫无必要的三级 Hive pipeline 简化成单个 MR, 把实现得又复杂又烂的 Hadoop + Hive 换成了 Spark 好得多的 MapReduce 计算模型实现。文中列出的对 Spark 的改进，正是一点点抠性能的工作。面对数据规模的挑战，你就是一个苦逼的系统工程师，不是住三藩用 Python 的数据科学家。

Rob Pike 操刀的 [Sawzall](http://static.googleusercontent.com/media/research.google.com/en//archive/sawzall-sciprog.pdf) 是一个构建于 MapReduce 模型之上的专用于 log 和数据分析的语言。例如前文的销售分析任务 —— MH 只是其中一个环节：

    proto “transaction.proto”;

    exp_sale: table sum[exp: int] of {int, int};
    prod_exp: table sum[prod_id: string][exp: int] of {int, int};
    price_dist: table quantiles[prod_id: string] of int;

    txn: Transaction = input
    emit price_dist[txn.prod_id] <- txn.price;
    for i := 0; i < len(txn.experiments); i++ {
      emit exp_sale[txn.experiments[i]] <- {txn.price, 1} ;
      emit prod_exp[txn.prod_id][txn.experiments[i]] <- {txn.price, 1};
    }

我对它的第一印象就是丑（觉得 Go 丑的，来看看 Sawzall，R 先生这些年也是有进步的好伐），而且功能也很受限, 我无法理解为何它能成为 Google 长期官方指定 log 处理语言。直到真正折腾过一些在 Google 也算是大的数据集，我才品出了它的一些味道。

Sawzall 提供的不是类似 SQL 的高层次抽象，而是一个类 C 的过程式语言 —— 用户写的程序运行在 map 节点，input 就是 map 的输入，用 table 去定义 reduce 的逻辑, 用 emit 输出到 reduce 节点。采用这个设计是因为它最常面对的原始 log 数据都是很「脏」的，这一步的数据处理有大量的工作就是用各种龌龊的逻辑洗数据，过程式语言其实更契合这种需要。

Sawzall 只能表达单个 MR, 却有定义多个 table —— 也就是 reduce 阶段多种不同 Reducer 逻辑的能力，上文中的例子就在一个 MR 中计算了三个结构完全不同的聚合表。这非常实用，比如还我可以随时再加上一个按卖家聚合的表，或者加上异常交易过滤，并且把异常交易打到另一张不聚合的表上。而且，这些特性都是保证在一趟 MR 中完成的，很多我以为需要更复杂的数据流的任务，在 Sawzall 看似简陋的框架下，不依赖高级而行为不确定的优化 planner, 却仍有最简也最优的实现。

有趣的是，Reduce 逻辑是不可定制的，你只能从标准的 sum, unique 之类的算子里选择。这看上去很不灵活，但再仔细想想，这是一个很厉害的观察：在数据处理中，map 部分不同业务自然各不相同，但 reduce 部分却通常是业务无关的统计函数，差异不大。更重要的是，reduce 部分通常是系统性能的瓶颈，标准最优化实现比灵活可定制重要。高效实现这些统计函数比看起来的要难：你要怎么实现 Percentile? 不会真的把所有值发到 reduce 节点死算吧？（[提示](http://infolab.stanford.edu/~datar/courses/cs361a/papers/quantiles.pdf)）所以干脆把他们做成基本算子，在 Sawzall 语言之外直接用 C++ 提供最佳实现。在这类任务最常见瓶颈上，Sawzall 就已经预先做好了优化。

不过话说回来，要实现这些能力并不需要一门独立语言，做个 C++ 库就可以了。Log 分析需要独立语言的主要原因是权限控制和审计（参见[文章](http://www.unofficialgoogledatascience.com/2015/12/replacing-sawzall-case-study-in-domain.html)），却也帮助了另一项重要性能优化：和它其他一切数据一样，Google 的 log 并不是文本，而每条都是一个硕大无朋的 protobuf, 里面事无巨细记下了事务的一切细节（听说最近这又叫 structual logging 了），一个分析脚本通常只会用到这其中的一小部分数据。Sawzall 被设计成一个很小的类型安全的语言，可以很容易做静态分析去发现程序到底读了输入 protobuf 的哪些部分，对于没有用到的部分，框架在解析输入 protobuf 的时候就可以直接跳过。大部分的 map 数据处理虽然业务逻辑复杂，算法却简单，相形之下，解析 protobuf 占 CPU 时间的比重很大，这一个优化对性能的帮助就大大超过 Sawzall 相比 C++ 的性能损失。

Protobuf 解析也许是 Google 特有的问题，Sawzall 的静态分析在 PL 上也没什么创见；再往大了说，这种 table 抽象也不是什么开创性的成就，把 aggregator 往死里优化也只是民工活。Sawzall 在外的名声远比不上 Flume 或 Dremel 等后辈，但在我眼中，提供实用的抽象的同时保持执行框架最简单，不贪图特性，不放过任何性能瓶颈上的优化机会，这就是大数据的玩法。

一个后记，Sawzall 已经在 Google deprecated, 替代者是一个叫 Lingo 的 Go 库。Go 和 Sawzall 一样类型安全，能快速编译，做静态分析与特定优化，同时有好得多的速度和组织更大规模程序的能力，Reducer 不再需要用另一种语言而是直接用 Go 实现，自然也就可以支持 customized reducer. 这也是一个有趣的故事，记录在 The Unofficial Google Data Science Blog 的[这篇文章](http://www.unofficialgoogledatascience.com/2015/12/replacing-sawzall-case-study-in-domain.html)里。本文所有材料都来自这个 unofficial blog 和其引用的公开材料，它的其他文章也非常有看头，严重推荐。
