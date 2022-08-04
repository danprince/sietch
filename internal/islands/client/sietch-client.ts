export function onIdle(): Promise<void> {
  return new Promise(resolve => {
    requestIdleCallback(() => resolve());
  })
}

export function onVisible(element: HTMLElement): Promise<void> {
  return new Promise(resolve => {
    let observer = new IntersectionObserver(entries => {
      if (entries[0].isIntersecting) {
        resolve();
        observer.disconnect();
      }
    }, { threshold: [0] });

    observer.observe(element);
  })
}
