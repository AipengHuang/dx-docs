import assert from 'node:assert/strict'
import test from 'node:test'

import { sanitizeUserVisibleBrandText } from './productBrand.ts'

test('rebrands legacy product names in user-visible text case-insensitively', () => {
  assert.equal(
    sanitizeUserVisibleBrandText('WeKnora / weknora / WEKNORACLOUD'),
    '帝显 / 帝显 / 帝显CLOUD',
  )
})

test('preserves ordinary assistant content', () => {
  assert.equal(sanitizeUserVisibleBrandText('你好，我是帝显。'), '你好，我是帝显。')
})
