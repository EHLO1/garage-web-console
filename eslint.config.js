import js from '@eslint/js';
import globals from 'globals';
import svelte from 'eslint-plugin-svelte';
import tseslint from 'typescript-eslint';
export default tseslint.config(
  { ignores: ['dist/**', '.svelte-kit/**', 'backend/ui/dist/**'] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  ...svelte.configs['flat/recommended'],
  {
    languageOptions: { globals: { ...globals.browser, ...globals.node } },
    rules: { '@typescript-eslint/no-explicit-any': 'off' }
  },
  {
    files: ['**/*.svelte', '**/*.svelte.ts'],
    languageOptions: { parserOptions: { parser: tseslint.parser } },
    rules: {
      'svelte/no-navigation-without-resolve': 'off',
      'svelte/prefer-svelte-reactivity': 'off'
    }
  }
);
