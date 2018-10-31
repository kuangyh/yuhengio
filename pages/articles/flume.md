@title Flume

# Flume

我已经盯着这个 Flume 程序发了半个小时呆。

[Flume](http://static.googleusercontent.com/media/research.google.com/en//pubs/archive/35650.pdf) 的公开版本叫 Dataflow, 确实是个好东西。它有个类似 LINQ 或者 Spark 的编程接口，你只要像组织 SQL 一样，写几个 ParDo, GroupBy 和 Join, 框架的 planner 会自动帮你转化成几个互相依赖的 MapReduce 任务，完成很复杂的数据计算。

我眼前这个程序，粗粗算来，为了个简单的把几个数据源聚合到一块的功能，它要跑 6 趟 MapReduce, 编译来一跑，果然不差。

6 趟。

Flume 很强悍的，再复杂点，搞个十几趟它都面不改色给你跑出来。更重要的是，这个编程模型太好用了，我看了下代码历史，最近一次改动轻描淡写地加了个小数据源，加了个 GroupBy & Join, 嗯，又加了一趟 MR.

这段程序写得还真不错，结构标准，还把一些数据接口封装了共用库，一行代码，你就可以要到很复杂的数据处理结果，一行代码，你就跑了 6 趟 MR. 这个项目十几个 daily jobs, 每天都这么跑上几个钟头。

我的胃开始疼了。

我想了想，调整一下顺序，Flume 应该能把其中一些步骤并行起来，这要看 planner 有多聪明了，人要顺着机器的脾性，从单机数据库到 Spanner / F1 再到 Flume 我习惯了。但是如果裸奔 MR, 我确定能全程两趟跑完。再说了，那么多 jobs 依赖这个数据源，怎么着也应该先跑个 job 把数据聚合好 dump 下来，别的 job 再来调吧？

不过，这么一来，再做改动就没法轻描淡写了，你要想怎么融入现有流程避免再加 MR. 要用一个 dump 下来的共用数据源就又有数据依赖问题，调度问题，版本同步和兼容问题一大堆。或者简单说，这不够 scalable.

据说这些都不再应该是工程师要烦恼的问题，MapReduce is deprecated, 我们已经有了新的一层抽象，你应该专注你的数据，你的分析，别再来计较这跑了几趟 MR.

然而，6 趟 MapReduce 的阴影依然挥之不去。

我见过另一个极端，每个模块都为了自己那一点点特殊自己写一个 thread pool 的，觉得多分配几次内存要死的。不过话说回来，你猜猜看有多少服务器 CPU 全耗在 memory allocator 上了？

要说大道理很容易了，没做过 profile 的性能优化都是耍流氓啦，一切都是 trade-off 啦。但每一个具体问题，都还是如此新鲜有趣。

Bjarne Stroustrup 给「什么是系统编程（Systems Programming)」下了 [这么个定义](https://channel9.msdn.com/Events/Lang-NEXT/Lang-NEXT-2014/Panel-Systems-Programming-Languages-in-2014-and-Beyond): 如果你要解决的问题遇上了显著的（硬件）资源限制，那你就要进入系统编程领域了。

那么，任何一个足够大，足够难的工程问题，最后都会或多或少地进入 systems 领域，运用这个领域的思维方式。

这是一个很有趣的领域。
