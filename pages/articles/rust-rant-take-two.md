@date Jul 27, 2015

# 为什么我没事就黑 Rust

没错，我就是没事就要黑一黑 Rust. Rust 跟以 Node.js 为代表的一系列流行一时的技术一样，看起来犀利无比，让没经过真实工程修罗场的新手们趋之若鹜 —— 毫无疑问这些人也是社区里最积极，声音最大的人群。到他们的致命问题露出狰狞面孔时，这些缺乏经验的聪明人又要绞尽脑汁想 fix 的方法并以此自豪，就好像 Node.js 中前仆后继要 fix callback hell 的人们。唉，要早知道是坑，别往下跳就完了嘛。

Rust 的致命问题就在它的最大创新 ownership 系统，没到大规模应用，人们都不会发现它“比C++容易”的欺骗性，也不会体会到它的复杂性的严重后果 —— 他们应该从 Node.js 和 C++ 里吸取这些教训的。Rust 核心开发团队也早就开始了他们努力解决根本不需要出现的问题的进程。

不久前，我偶然翻到 Rust Ownership 系统主要设计者 Niko Matsakis 的 [blog](http://smallcultfollowing.com/babysteps/), 发现这基本就是自己用枪设自己的脚然后自己包扎的最佳案例。例如，原来 thread::scope 在 Rc 循环引用时会造成 destruct 后引用的大乌龙，是 [Niko 自己搞出来的](http://smallcultfollowing.com/babysteps/blog/2015/04/29/on-reference-counting-and-leaks/), 从 proposal 到实现这么长时间竟然没发现它跟自己设计的模型的矛盾，老兄你逗我玩？然后还有他 [处理图里乱指的指针](http://smallcultfollowing.com/babysteps/blog/2015/04/06/modeling-graphs-in-rust-using-vector-indices/) 的办法，嗯，就是用数组一存然后用 index 指，也就是，在 Rust 中重新发明了自由指针。然后他还重新思考了 dangling pointer 问题，然后说，in practice 这没啥问题，比纯指针安全多了 —— in practice 你学好基础知识遵守编码规范好好用 C++ 也不会有啥问题，那你说你的 onwership 系统有啥用呢？你 dangaling index 把这个用户的私信指到另一个人上然后辩解说“这不是内存安全问题Rust还是安全的” 就是真的然并卵啊。

所以我要不遗余力地黑 Rust 这样的坑，它太能迷惑人，没足够的经验，甚至都没法发现这是坑。但是，少点人掉坑，就减少点资源浪费。已掉坑的，不管是 Node.js 还是 Rust 的，大概都会觉得我这儿是一派胡言，要把掉坑的拉回上来，也挺浪费资源的。
