// eslint-config-next@16 ships native flat-config arrays (Linter.Config[]) per
// sub-path export, so we import them directly instead of routing through
// `FlatCompat.extends('next/core-web-vitals', 'next/typescript')` — that
// legacy bridge expects eslintrc-shaped configs and throws
// "Converting circular structure to JSON" against the plugin objects that
// v16's flat exports already resolve. See ADR-008 §3 deviation note.
import nextCoreWebVitals from 'eslint-config-next/core-web-vitals';
import nextTypescript from 'eslint-config-next/typescript';

const config = [
  { ignores: ['.next/**', 'node_modules/**', 'next-env.d.ts'] },
  ...nextCoreWebVitals,
  ...nextTypescript,
];

export default config;
