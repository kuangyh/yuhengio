@date Oct 24, 2015

# iOS 开发一月记

## Swift

Swift 不能算是一门独立的新语言，我把它看成一个卸掉很多包袱的现代版 Objective-C —— 这有点像是 Scala 之于 Java. 没有 Objective-C 多年积累下来的 SDK、库和一个稳定的运行时环境，Swift 是没法玩得转的。

对我来说，Objective-C 的设计本来就不错，而 Swift 因为没有甚至要跟 C 兼容的历史包袱，可以免掉很多不知所云的控制符和罗嗦丑陋的语法，也可以做更多正确的语言选择，例如终于成为值类型和不需要 boxing 的容器类，更多的静态派发但保留动态性（跟 Java 的故事好像）。学习 Swift 的 “Aha moment” 依然是在搞清楚各个关键元素的实现方式，对应到有良好文档的 Objective-C 运行时（不过注意他们不是同一个东西）的时候 —— 很庆幸我上一次密集写 Objective-C 程序的时候它连 ARC 都还没有，逼得我先把它的内部模型弄熟，这次在 Swift 上就省事了。

Swift 编译器和 SourceKit 现在还是有点小 bug, 从复杂点的类型推断摊手无力（Go 开发者又会冷笑，谁叫你加 generics 了，但 Swift 的 generic 已经属于最轻量，副作用最小，性价比最高的了）到 compiler crash 不一而足。这方面 Apple 的改进速度很快，毫不担心。

## Storyboard

看来 Apple 对自己的开发者也喜欢玩 “It magically happens” 这一套，说难听点就叫愚民政策，不断弄出各种漂亮的工具让 iOS 开发变得更傻瓜。结果，你已经很难从头到尾解释你的程序是怎么运行起来的，就更难调错和优化了。

所以我选择了先丢开 Storyboard 等 fancy 工具，直接全部代码开发，疯狂查文档。这样程序的每一个行为，我都是可以对应到代码，也有文档可以解释的。头一两天的不适应之后，学习曲线就平滑起来，能够很快做有效的开发，在其他平台的开发经验也能用上，有问题也能自己调出来。

抛开开发者技术背景和项目要求谈开发技术都是耍流氓。如果开发的应用都是 Apple 的标准控件标准布局，那 Storyboard 和 Interface Builder 自然是最方便的，按着教程来就行。如果团队里有非常熟悉 iOS 开发，不管出现什么问题都兜得住的小叮当类型人物，那 Storyboard 也能提高开发速度。但如果对这套环境还没熟悉到这个程度，却又要开发交互复杂、定制程度高的界面，那还是只用可控性好的开发技术比较保险。

## Design Patterns

论 MVC design pattern, Cocoa / UIKit 直接继承 Smalltalk 的衣钵，可谓根正描红。可是搞开发，血统正是木有用的。我生而有幸，先经历学习 JavaScript 的最好办法是自己写个 MVC framework 的时代，再在原产地经历 GWT 和 Angular 的双重摧残，再回头看 iOS 的 MVC, 就少了点神圣感，而多了点批判的眼光。

具体的吐槽 Facebook 就[帮我吐完了](https://www.youtube.com/watch?v=mLSeEoC6GjU), 其实就两点，MVC 对 Model 的假设是数据全部来自本地存储，也将被同步到本地存储，所以一个 Core Data + KVO 就能满足你的所有愿望，但是如果你做的是一个互联网应用的客户端，这个假设就不但不成立，还阻手碍脚，因为你需要考虑两倍的同步问题（Apple 会说谁叫你不用 iCloud），因此你最后会分离出一个 service 层，或者叫 Store 的单元，去处理网络数据获取、磁盘同步（此时本地存储相当于 cache) 等问题 —— 这些问题 Web 开发社区经历得够多了；另一个问题就是基于 KVO 的双向 binding 是不 scalable 的，一开始你会觉得 ah! magic! 应用稍微复杂一点那就是一团乱麻，Web 社区早期的 MVC 框架 backbone 也主打双向 binding, 到 Angular 也依然是个卖点，但现在的 Angular 社区和 Polymer 已经推荐使用单向 binding. 更不用说大受欢迎的 React 框架了。

说到 React, 如果放到一个大背景上看，就是 UI 开发（不管是 Web 还是 Native）里，functional programming 逐渐进入了主流。我经常说 FP 社区每隔几年就要出来碾压一次工业界，最让人郁闷的是，**他们说的都是对的！** 这一次，主要是 Haskell 出来鄙视全世界了。 Swift 里更好的 closure 支持和 optional chain（让世界充满 Monad）之类都还只能算小甜点了，Promise 算是小试牛刀（理解难度比 monad 低了 N 个数量级，嘿嘿），现在正当红的是 pure functional 的 React 和正牌 reactive programming 的 ReactiveCocoa。这都是 Haskell 玩了多少年的东西了，现在它们出现在主流工业界，主要原因看来还是 async IO 成为每个做 UI 的人必须考虑的问题，在异步的世界，时序和状态维护的复杂性终于显现出来，那群傻傻地坚持 pure functional 的 haskell 人民多年来研究的黑科技，也就终于有了用武之地。

我会写 Hello World 以来，遇上的最大的两个概念难关，一个是看 SICP 时折腾 closure 和 continuation, 另一个就是 Haskell 里的 Monad. 玩转了 Scheme 之后再去看 JavaScript, 旁人觉得再高级再难懂的用法都是小菜一碟，而当年在 monad 上花费的不眠之夜，现在也开始 pay off 了。计算机真的是一个有完整理论体系的学科，打好基础，在工业界能够轻松许多。

## Community

Apple 无比完整还不断更新的基础库的一个后果，就是开源项目通常不用做太深太难的事情，常常就是把 Apple 的一些 API wrap 一下。于是你不会遇上 C++ 或者 PHP 这种特别开放的社区的常见问题：不管做啥都得先找社区实现，一个功能的实现至少有互不兼容的一打，各有各的互不兼容的依赖和缺陷，没有一个质量过关。iOS 的开源项目通常比较简单，质量也不会有什么大问题（因为没有难的代码），而且，读 code 很容易。

我一直相信学习一个开发平台和一项技术的最好方法就是读代码，了解他们到底是怎么运作的。有的语言或开发平台的代码非常难读：比如 C 和 C++，每个作者都会自己 wrap 一套 macro 库和资源管理系统，都有自己的编码规范，每看一个新项目都要怀疑自己到底懂不懂 C / C++, LISP 那些就是自己跟自己过不去了。但 Objective-C 的代码就好了很多，而读 Swift 的代码，就是跟读 Go 代码一样的享受。
