const fs = require('fs');
const path = require('path');
const postcss = require('postcss');

const webDir = path.resolve(__dirname, '..');
const cssPath = path.join(webDir, 'src/styles.css');
const outDir = path.join(webDir, 'src/styles');
const pagesDir = path.join(webDir, 'src/pages');

const pageFiles = fs.readdirSync(pagesDir).filter((f) => f.endsWith('.tsx'));
const pageSource = new Map(pageFiles.map((f) => [f, fs.readFileSync(path.join(pagesDir, f), 'utf8')]));

const TARGETS = {
  'station.css': ['Station.tsx'],
  'model.css': ['ModelPage.tsx'],
  'models.css': ['Models.tsx'],
  'stations.css': ['Stations.tsx'],
  'compare.css': ['Compare.tsx', 'ModelCompare.tsx'],
  'status.css': ['Status.tsx', 'ModelStatus.tsx'],
  'leaderboards.css': ['Leaderboards.tsx'],
  'audit.css': ['Audit.tsx'],
  'free.css': ['Free.tsx', 'FreeKeys.tsx'],
  'admin.css': ['Admin.tsx'],
  'changes.css': ['Changes.tsx'],
  'home.css': ['Home.tsx'],
};

const classPages = new Map();
const classRe = /\.(-?[_a-zA-Z][_a-zA-Z0-9-]*)/g;
const cache = new Map();
function pagesUsing(cls) {
  if (cache.has(cls)) return cache.get(cls);
  const re = new RegExp(`(^|[^\\w-])${cls.replace(/[-]/g, '\\-')}([^\\w-]|$)`);
  const hits = new Set();
  for (const [file, src] of pageSource) if (re.test(src)) hits.add(file);
  cache.set(cls, hits);
  return hits;
}

const css = fs.readFileSync(cssPath, 'utf8');
const root = postcss.parse(css);

const buckets = new Map(Object.keys(TARGETS).map((f) => [f, []]));

function targetFor(selectors) {
  const union = new Set();
  const classes = new Set();
  for (const sel of selectors) {
    let m;
    classRe.lastIndex = 0;
    let found = false;
    while ((m = classRe.exec(sel))) {
      found = true;
      classes.add(m[1]);
    }
    if (!found) return null;
  }
  for (const cls of classes) {
    const pages = pagesUsing(cls);
    if (pages.size === 0) return null;
    for (const p of pages) union.add(p);
  }
  for (const [file, allowed] of Object.entries(TARGETS)) {
    if (union.size > 0 && [...union].every((p) => allowed.includes(p))) return file;
  }
  return null;
}

root.walkRules((rule) => {
  let p = rule.parent;
  while (p && p.type === 'atrule' && /keyframes/.test(p.name)) return;
  const sels = rule.selectors || [rule.selector];
  const file = targetFor(sels);
  if (!file) return;
  const media = rule.parent && rule.parent.type === 'atrule' && /media/.test(rule.parent.name) ? rule.parent.params : null;
  buckets.get(file).push({ css: rule.toString(), media, rule });
});

let total = 0;
for (const [file, items] of buckets) {
  if (!items.length) continue;
  let out = '';
  for (const item of items) {
    out += item.media ? `@media ${item.media} {\n${item.css}\n}\n` : item.css + '\n';
    item.rule.remove();
    total += 1;
  }
  fs.mkdirSync(outDir, { recursive: true });
  fs.writeFileSync(path.join(outDir, file), out.trimStart() + '\n');
  console.log(file, items.length, 'rules');
}

root.walkAtRules((at) => {
  if (/media/.test(at.name) && at.nodes && at.nodes.length === 0) at.remove();
});
fs.writeFileSync(cssPath, root.toString());
console.log('total moved', total);
