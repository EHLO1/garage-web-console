import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { defineConfig, loadEnv } from 'vite';
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd());
  return {
    plugins: [
      sveltekit({
        preprocess: vitePreprocess(),
        adapter: adapter({
          pages: 'dist',
          assets: 'dist',
          fallback: 'index.html'
        }),
        files: { assets: 'public' },
        // Go replaces this marker at runtime; the same binary supports any BASE_PATH.
        paths: {
          base: process.env.NODE_ENV === 'production' ? '/__garage_base__' : '',
          relative: false
        }
      })
    ],
    server: {
      proxy: {
        '/api': {
          target: env.VITE_API_URL || 'http://localhost:3909',
          changeOrigin: true
        }
      }
    }
  };
});
