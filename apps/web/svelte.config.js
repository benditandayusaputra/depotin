import adapter from '@sveltejs/adapter-vercel';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

const streamOrigin = process.env.PUBLIC_STREAM_ORIGIN ?? 'http://localhost:8080';

const config = {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter({ runtime: 'nodejs22.x', regions: ['sin1'] }),
    csp: {
      mode: 'hash',
      directives: {
        'default-src': ['self'],
        'script-src': ['self'],
        'style-src': ['self', 'unsafe-inline'],
        'img-src': ['self', 'data:'],
        'connect-src': ['self', streamOrigin],
        'frame-ancestors': ['none'],
        'base-uri': ['self'],
        'form-action': ['self']
      }
    }
  }
};

export default config;
