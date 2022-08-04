import { useState } from "preact/hooks";

export default ({ count: init = 0 }) => {
  let [count, setCount] = useState(init);
  return <button onClick={() => setCount(count + 1)}>{count}</button>;
}
