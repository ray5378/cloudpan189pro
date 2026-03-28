declare module 'plyr' {
  // 轻量类型声明：提供类与构造签名以满足 TS 对值与类型的双重使用
  export default class Plyr {
    constructor(element: HTMLElement | HTMLMediaElement, options?: Record<string, unknown>)
    destroy(): void
  }
}
