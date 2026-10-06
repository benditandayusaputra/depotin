import js from '@eslint/js';
import svelte from 'eslint-plugin-svelte';
import globals from 'globals';
import ts from 'typescript-eslint';
import svelteConfig from './svelte.config.js';

export default ts.config(
  { ignores: ['.svelte-kit/', 'build/', 'node_modules/', '.vercel/', 'src/lib/api/schema.d.ts'] },
  js.configs.recommended,
  ...ts.configs.recommended,
  ...svelte.configs.recommended,
  ...svelte.configs.prettier,
  { languageOptions: { globals: { ...globals.browser, ...globals.node } } },
  {
    files: ['**/*.svelte', '**/*.svelte.ts'],
    languageOptions: {
      parserOptions: {
        projectService: true,
        extraFileExtensions: ['.svelte'],
        parser: ts.parser,
        svelteConfig
      }
    }
  },
  { files: ['src/app.d.ts'], rules: { '@typescript-eslint/no-empty-object-type': 'off' } },
  { rules: { 'svelte/no-at-html-tags': 'error', '@typescript-eslint/no-explicit-any': 'error' } }
);
