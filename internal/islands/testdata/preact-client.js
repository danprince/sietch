import { onIdle, onVisible } from '@sietch/client';
import { h, hydrate } from 'preact';

import $ca from './Counter.tsx';
hydrate(h($ca, {"count":1}), document.getElementById('a'));

onIdle()
  .then(() => import('./Counter.tsx'))
  .then(md => hydrate(h(md.default, {"count":3}), document.getElementById('b')));

let $ec = document.getElementById('c');
onVisible($ec)
  .then(() => import('../Timer.tsx'))
  .then(md => hydrate(h(md.default, {}), $ec));
