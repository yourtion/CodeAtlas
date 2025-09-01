import { defineConfig } from '@rsbuild/core';
import { pluginSvelte } from '@rsbuild/plugin-svelte';

export default defineConfig({
  plugins: [
    pluginSvelte()
  ],
  server: {
    port: 3000,
  },
  dev: {
    assetPrefix: '/',
  },
  output: {
    assetPrefix: '/',
  },
  source: {
    entry: {
      index: './src/main.js',
    },
  },
});