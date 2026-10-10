const fs = require('fs');
let file = fs.readFileSync('frontend/pages/meta-pages.vue', 'utf8');
file = file.replace(
  /<v-img v-if="pictureAvailable\(page\)" :src="pagePictureUrl\(page\)" :alt="`\$\{page\.name\} profile picture`" cover @error="handlePictureError\(page\.id\)"><\/v-img>/g,
  '<img v-if="pictureAvailable(page)" :src="pagePictureUrl(page)" :alt="`${page.name} profile picture`" referrerpolicy="no-referrer" @error="handlePictureError(page.id)" style="width: 100%; height: 100%; object-fit: cover;">'
);
fs.writeFileSync('frontend/pages/meta-pages.vue', file);
