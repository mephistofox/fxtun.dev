import { test } from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { parseFrontmatter, renderDoc, slugify, validateManifest } from './docs-md.ts'

const doc = (body: string) =>
  `---\ntitle: Заголовок страницы\ndescription: Описание страницы\n---\n${body}`

test('frontmatter: поля и тело', () => {
  const { data, body } = parseFrontmatter(doc('Текст\n'), 'a.md')
  assert.equal(data.title, 'Заголовок страницы')
  assert.equal(data.description, 'Описание страницы')
  assert.equal(body, 'Текст\n')
})

test('frontmatter: нет description — ошибка с именем файла', () => {
  assert.throws(() => parseFrontmatter('---\ntitle: T\n---\nx', 'b.md'), /b\.md: во frontmatter нет description/)
})

test('frontmatter: нет блока — ошибка', () => {
  assert.throws(() => parseFrontmatter('# Текст', 'c.md'), /c\.md: нет frontmatter/)
})

test('slugify сохраняет кириллицу и схлопывает знаки', () => {
  assert.equal(slugify('Свой поддомен'), 'свой-поддомен')
  assert.equal(slugify('Basic Auth'), 'basic-auth')
  assert.equal(slugify('`--auth` флаг'), 'auth-флаг')
})

test('h2/h3 получают id, якорь и попадают в headings; повторы с суффиксом', () => {
  const r = renderDoc(doc('## Свой поддомен\n\n### `--auth` флаг\n\n## Свой поддомен\n'), 'd.md')
  assert.deepEqual(r.headings, [
    { level: 2, id: 'свой-поддомен', text: 'Свой поддомен' },
    { level: 3, id: 'auth-флаг', text: '--auth флаг' },
    { level: 2, id: 'свой-поддомен-2', text: 'Свой поддомен' },
  ])
  assert.match(r.html, /<h2 id="свой-поддомен">Свой поддомен<a class="doc-anchor" href="#свой-поддомен"/)
})

test('h1 в тексте запрещён', () => {
  assert.throws(() => renderDoc(doc('# Заголовок\n'), 'e.md'), /e\.md: заголовок первого уровня/)
})

test('raw HTML не проходит', () => {
  const r = renderDoc(doc('<script>alert(1)</script>\n'), 'f.md')
  assert.ok(!r.html.includes('<script>'))
  assert.ok(r.html.includes('&lt;script&gt;'))
})

test('shell-блок: приглашение, флаги, вывод, комментарий и текст для копирования', () => {
  const r = renderDoc(
    doc('```bash\nfxtunnel http 3000 --domain myapp\n→ https://myapp.fxtun.ru\n# комментарий\n```\n'),
    'g.md',
  )
  assert.ok(r.html.includes('<span class="term-lang">bash</span>'))
  assert.ok(r.html.includes('<span class="t-ps">$</span>fxtunnel http 3000 <span class="t-flag">--domain</span> myapp'))
  assert.ok(r.html.includes('<span class="t-out">→ https://myapp.fxtun.ru</span>'))
  assert.ok(r.html.includes('<span class="t-cm"># комментарий</span>'))
  assert.ok(r.html.includes('data-copy="fxtunnel http 3000 --domain myapp\n# комментарий"'))
})

test('shell-блок: продолжение строки после \\ без приглашения', () => {
  const r = renderDoc(doc('```bash\nfxtunnel http 3000 \\\n  --auth admin:StrongPass123\n```\n'), 'h.md')
  assert.equal(r.html.split('class="t-ps"').length - 1, 1)
  assert.ok(r.html.includes('<span class="t-flag">--auth</span>'))
})

test('не-shell блок только экранируется', () => {
  const r = renderDoc(doc('```yaml\nkey: <value>\n```\n'), 'i.md')
  assert.ok(r.html.includes('<span class="term-lang">yaml</span>'))
  assert.ok(r.html.includes('key: &lt;value&gt;'))
  assert.ok(!r.html.includes('t-ps'))
})

test('выноска: маркер на своей строке', () => {
  const r = renderDoc(doc('> [!TIP]\n> Текст совета\n'), 'j.md')
  assert.ok(r.html.includes('<div class="callout callout-tip">'))
  assert.ok(r.html.includes('<b>Совет</b>'))
  assert.ok(r.html.includes('Текст совета'))
  assert.ok(!r.html.includes('[!TIP]'))
  assert.ok(!r.html.includes('<blockquote>'))
})

test('выноска: маркер в начале строки с текстом', () => {
  const r = renderDoc(doc('> [!WARNING] Осторожно\n'), 'k.md')
  assert.ok(r.html.includes('<div class="callout callout-warn">'))
  assert.ok(r.html.includes('<b>Внимание</b>'))
  assert.ok(r.html.includes('Осторожно'))
})

test('обычная цитата остаётся цитатой', () => {
  const r = renderDoc(doc('> Просто цитата\n'), 'l.md')
  assert.ok(r.html.includes('<blockquote>'))
})

test('таблица обёрнута для скругления и прокрутки', () => {
  const r = renderDoc(doc('| a | b |\n|---|---|\n| 1 | 2 |\n'), 'm.md')
  assert.ok(r.html.includes('<div class="doc-table"><table>'))
})

test('время чтения без блоков кода, минимум минута', () => {
  const words = Array.from({ length: 360 }, () => 'слово').join(' ')
  assert.equal(renderDoc(doc(`${words}\n\n\`\`\`bash\n${words}\n\`\`\`\n`), 'n.md').minutes, 2)
  assert.equal(renderDoc(doc('Коротко.\n'), 'o.md').minutes, 1)
})

function docsDir(manifest: unknown[], files: string[]): string {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'fxtun-docs-'))
  fs.writeFileSync(path.join(dir, 'manifest.json'), JSON.stringify(manifest))
  for (const f of files) fs.writeFileSync(path.join(dir, f), doc('Текст\n'))
  return dir
}

const entry = (slug: string, extra: object = {}) => ({ slug, group: 'Туннели', nav: 'Пункт', lastmod: '2026-09-23', ...extra })

test('манифест: корректный возвращает записи', () => {
  const dir = docsDir([entry(''), entry('http', { tag: 'http' })], ['index.md', 'http.md'])
  assert.equal(validateManifest(dir).length, 2)
})

test('манифест: нет файла', () => {
  assert.throws(() => validateManifest(docsDir([entry('http')], [])), /http: нет файла http\.md/)
})

test('манифест: лишний .md', () => {
  assert.throws(() => validateManifest(docsDir([entry('')], ['index.md', 'tcp.md'])), /tcp\.md: нет в manifest\.json/)
})

test('манифест: повтор slug', () => {
  assert.throws(() => validateManifest(docsDir([entry('tcp'), entry('tcp')], ['tcp.md'])), /tcp: slug повторяется/)
})

test('манифест: зарезервированный и кривой slug', () => {
  assert.throws(() => validateManifest(docsDir([entry('offer')], ['offer.md'])), /offer: slug занят/)
  assert.throws(() => validateManifest(docsDir([entry('Http')], ['Http.md'])), /Http: slug только из a-z/)
})

test('манифест: обязательные поля, дата, tag', () => {
  assert.throws(() => validateManifest(docsDir([entry('udp', { nav: '' })], ['udp.md'])), /udp: нет nav/)
  assert.throws(() => validateManifest(docsDir([entry('udp', { lastmod: '23.09.2026' })], ['udp.md'])), /udp: lastmod/)
  assert.throws(() => validateManifest(docsDir([entry('udp', { tag: 'quic' })], ['udp.md'])), /udp: tag только http, tcp или udp/)
})
