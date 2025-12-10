/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 *
 * @author ...
 */

import vue from 'eslint-plugin-vue';

export default [
	{
		ignores: [
			"dist/**",
			"public/**",
			"node_modules/**",
		],
	},
	...vue.configs["flat/recommended"],
	{
		rules: {
			'vue/multi-word-component-names': 'off',
			'vue/max-attributes-per-line': 'off',
			'vue/require-default-prop': 'off',
			'vue/singleline-html-element-content-newline': 'off',
		},
	},
];
