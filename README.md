# hackpad

Hackpad is a [Go][] development environment with the essentials to write and run code entirely within the browser, using the power of [WebAssembly (Wasm)][wasm].

Check out the article announcement on [Medium][], and the site at https://hackpad.org


[Go]: https://golang.org
[wasm]: https://webassembly.org
[Medium]: https://johnstarich.medium.com/how-to-compile-code-in-the-browser-with-webassembly-b59ffd452c2b

## Contributing

Want to discuss an idea or a bug? Open up a new [issue][] and we can talk about it. Pull requests are welcome.

[issue]: https://github.com/hack-pad/hackpad/issues


## Known issues
* Slow compile times - Rewrite runtime to [parallelize with Web Workers](https://github.com/hack-pad/hackpad/issues/11)
* Safari crashes - Regularly crashes due to Wasm memory bugs. [WebKit #222097](https://bugs.webkit.org/show_bug.cgi?id=222097), [#227421](https://bugs.webkit.org/show_bug.cgi?id=227421), [#220313](https://bugs.webkit.org/show_bug.cgi?id=220313)
