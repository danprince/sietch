import { onIdle, onVisible } from '@sietch/client';

import { hydrate as $ha } from './Counter.tsx';
$ha({"count":1}, document.getElementById('a'));

onIdle()
  .then(() => import('./Counter.tsx'))
  .then(md => md.hydrate({"count":3}, document.getElementById('b')));

let $ec = document.getElementById('c');
onVisible($ec)
  .then(() => import('../Timer.tsx'))
  .then(md => md.hydrate({}, $ec));
