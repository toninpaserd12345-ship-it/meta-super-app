const fs = require('fs');
let file = fs.readFileSync('frontend/pages/meta-pages.vue', 'utf8');
file = file.replace(
  /const pagePictureUrl=\(page:MetaPage\)=>.*?`/g,
  `const pagePictureUrl=(page:MetaPage)=>page.category.toLowerCase().includes('whatsapp') ? (page.pictureUrl||'') : \`https://graph.facebook.com/v20.0/\${page.id}/picture?type=normal\`;//\``
);
fs.writeFileSync('frontend/pages/meta-pages.vue', file);
