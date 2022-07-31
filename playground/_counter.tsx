// @ts-nocheck
import "./_counter.css";
import { useState } from "preact/hooks";

let Counter = ({ count: initialCount = 0 }) => {
  let [count, setCount] = useState(initialCount);
  return (
    <button onClick={() => setCount(count + 1)}>{count}</button>
  );
}

export default Counter;
