export function Greet(name) {
  return Promise.resolve("Hello " + name + ", It's show time!");
}

export function ListSessions() {
  return window['go']['main']['App']['ListSessions']();
}
