const fs = require('fs');
let file = fs.readFileSync('frontend/pages/meta-pages.vue', 'utf8');

file = file.replace(
  /Pause Webhook for “\$\{page\.name\}”\? New events from this Page will stop until it is enabled again\./g,
  'ປິດການໃຊ້ງານສຳລັບ “${page.name}”? ຂໍ້ຄວາມໃໝ່ຈາກເພຈນີ້ຈະບໍ່ເຂົ້າລະບົບຈົນກວ່າຈະເປີດຄືນ.'
);

file = file.replace(
  /\$\{page\.name\}: Webhook \$\{enabled\?'enabled':'paused'\}\./g,
  '${page.name}: ${enabled?\'ເປີດການໃຊ້ງານສຳເລັດ\':\'ປິດການໃຊ້ງານສຳເລັດ\'}.'
);

file = file.replace(
  /activated\. Webhooks are ready\./g,
  'ບັນຊີ. ລະບົບພ້ອມຮັບຂໍ້ຄວາມແລ້ວ.'
);
file = file.replace(
  /\$\{successIds\.length\} Page\$\{successIds\.length===1\?'':'s'\} ບັນຊີ/g,
  'ເປີດການໃຊ້ງານສຳເລັດ ${successIds.length} ບັນຊີ'
);

file = file.replace(
  /WhatsApp connected\. The webhook is subscribed and ready to receive messages\./g,
  'ເຊື່ອມຕໍ່ WhatsApp ສຳເລັດ. ລະບົບພ້ອມຮັບຂໍ້ຄວາມແລ້ວ.'
);

file = file.replace(
  /ຈັດການບັນຊີ, Webhook ແລະຄວາມສຳພັນກັບ Automation ໃນບ່ອນດຽວ/g,
  'ຈັດການບັນຊີ, ການຕອບກັບ ແລະ ຕັ້ງຄ່າຂໍ້ຄວາມອັດຕະໂນມັດ (Automation) ໃນບ່ອນດຽວ'
);

file = file.replace(
  /WEBHOOK ACTIVE/g,
  'ເປີດນຳໃຊ້ແລ້ວ'
);

file = file.replace(
  /Webhook turns on automatically/g,
  'ລະບົບຈະເປີດຮັບຂໍ້ຄວາມອັດຕະໂນມັດ'
);

file = file.replace(
  /<strong>Webhook active<\/strong><small>Receiving events<\/small>/g,
  '<strong>ເປີດນຳໃຊ້ແລ້ວ</strong><small>ກຳລັງຮັບຂໍ້ຄວາມ</small>'
);

file = file.replace(
  /Pause Webhook for/g,
  'ປິດການໃຊ້ງານສຳລັບ'
);

file = file.replace(
  /Enable Webhook/g,
  'ເປີດການໃຊ້ງານ'
);

file = file.replace(
  /Page access, Webhooks and Messenger permissions are included automatically\. Select only the additional features you need\./g,
  'ສິດໃນການເຂົ້າເຖິງເພຈ ແລະ ການຮັບສົ່ງຂໍ້ຄວາມ ແມ່ນຖືກລວມໄວ້ແລ້ວອັດຕະໂນມັດ. ໃຫ້ທ່ານເລືອກສະເພາະສິດເພີ່ມເຕີມທີ່ຕ້ອງການໃຊ້ງານເທົ່ານັ້ນ.'
);

fs.writeFileSync('frontend/pages/meta-pages.vue', file);
