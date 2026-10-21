<script setup lang="ts">
definePageMeta({ middleware: 'auth' })
const { l } = useLocale()

const activeTab = ref<'routine' | 'metrics' | 'decision' | 'ghost'>('routine')
</script>

<template>
  <div class="ads-playbook-page">
    <!-- Top Hero Header -->
    <header class="playbook-hero">
      <div class="hero-left">
        <span class="hero-eyebrow"> MEDIA BUYER DAILY SOP & PLAYBOOK</span>
        <h1 class="hero-title">
          <span>ຄູ່ມືຍິງແອດປະຈຳວັນ</span>
          <span class="hero-badge">Standard Operating Procedure</span>
        </h1>
        <p class="hero-sub">
          {{ l({
            lo: 'ຂັ້ນຕອນການເຮັດວຽກໃນ 1 ມື້ຂອງຄົນຍິງແອດ, ວິທີອ່ານຕົວເລກ, ກົດການຕັດສິນໃຈ Kill/Hold/Scale ແລະ ວິທີກັ່ນຕອງທັກຜີ',
            th: 'ขั้นตอนการทำงานใน 1 วันของคนยิงแอด, วิธีอ่านตัวเลข, กฎการตัดสินใจ Kill/Hold/Scale และวิธีกรองทักผี',
            en: 'Daily media buyer routine, metric diagnostics, Kill/Hold/Scale decision tree, and ghost lead protocol'
          }) }}
        </p>
      </div>

      <div class="hero-actions">
        <NuxtLink to="/ads-battle">
          <v-btn color="primary" variant="flat" prepend-icon="mdi-scale-balance" class="font-weight-bold">
             ໄປໜ້າ Ads Battle
          </v-btn>
        </NuxtLink>
        <NuxtLink to="/live-chat">
          <v-btn variant="outlined" prepend-icon="mdi-chat-processing-outline">
             ໄປໜ້າ Live Chat
          </v-btn>
        </NuxtLink>
      </div>
    </header>

    <!-- Navigation Bar -->
    <nav class="playbook-nav">
      <button
        type="button"
        class="nav-tab-btn"
        :class="{ active: activeTab === 'routine' }"
        @click="activeTab = 'routine'"
      >
        <span class="tab-icon"></span>
        <span>1. ຕາຕະລາງປະຈຳວັນ (Daily Routine)</span>
      </button>

      <button
        type="button"
        class="nav-tab-btn"
        :class="{ active: activeTab === 'metrics' }"
        @click="activeTab = 'metrics'"
      >
        <span class="tab-icon"></span>
        <span>2. ວິທີອ່ານ 4 ຕົວເລກ (Diagnostics)</span>
      </button>

      <button
        type="button"
        class="nav-tab-btn"
        :class="{ active: activeTab === 'decision' }"
        @click="activeTab = 'decision'"
      >
        <span class="tab-icon"></span>
        <span>3. ກົດເຫຼັກຕັດສິນໃຈ (Kill, Hold, Scale)</span>
      </button>

      <button
        type="button"
        class="nav-tab-btn"
        :class="{ active: activeTab === 'ghost' }"
        @click="activeTab = 'ghost'"
      >
        <span class="tab-icon"></span>
        <span>4. ວິທີຈັດການ "ທັກຜີ" & ມືຫຼົງກົດ</span>
      </button>
    </nav>

    <!-- MAIN CONTENT -->
    <main class="playbook-body">
      <!-- ========================================== -->
      <!-- TAB 1: ຕາຕະລາງປະຈຳວັນ (Daily Routine) -->
      <!-- ========================================== -->
      <section v-if="activeTab === 'routine'" class="tab-section">
        <div class="section-header">
          <h2> ຕາຕະລາງປະຈຳວັນ: ມື້ໜຶ່ງຄວນເບິ່ງແອດຈັກເທື່ອ?</h2>
          <p class="section-lead">
            ກົດເຫຼັກ: <strong>ຢ່າເປີດເບິ່ງແອດທຸກໆ 10 ນາທີ</strong> ເພາະຈະເຮັດໃຫ້ທ່ານ "ໃຊ້ອາລົມ" ຕັດສິນ. ໃຫ້ເບິ່ງພຽງ 3 ຄັ້ງຕໍ່ມື້ຕາມຕາຕະລາງນີ້:
          </p>
        </div>

        <div class="time-cards-grid">
          <!-- Card 1: Morning -->
          <div class="time-card morning">
            <div class="time-badge morning">
              <span> 08:30 - 09:30 (ຕອນເຊົ້າ)</span>
              <strong>ກວດສອບສະພາບລວມ</strong>
            </div>
            <div class="card-content">
              <h4>ສິ່ງທີ່ຕ້ອງເຮັດ:</h4>
              <ul>
                <li>ເປີດເບິ່ງໜ້າ Ads Battle ເລືອກວັນທີ <strong>"ມື້ວານນີ້ (Yesterday)"</strong>.</li>
                <li>ເຊັກຍອດໃຊ້ຈ່າຍລວມ, ຈຳນວນອໍເດີ, ແລະ CPA ສະເລ່ຍຂອງມື້ວານ.</li>
                <li>ກວດສອບວ່າມີແອດໃດຕິດບັນຫາບໍ່ (ເຊັ່ນ ຖືກ Disapprove ຫຼື ບັດຕັດເງິນບໍ່ຜ່ານ).</li>
              </ul>
              <div class="rule-box warn">
                 <strong>ກົດຫ້າມຕອນເຊົ້າ:</strong> ຫ້າມປິດແອດທີ່ຫາກໍ່ເປີດໃໝ່ ຫຼື ປັບງົບຕອນເຊົ້າເດັດຂາດ! ເພາະແອດຫາກໍ່ເລີ່ມແລ່ນ ຄ່າແອດຈະຍັງແພງຢູ່.
              </div>
            </div>
          </div>

          <!-- Card 2: Afternoon -->
          <div class="time-card afternoon">
            <div class="time-badge afternoon">
              <span> 10:00 - 18:00 (ຕອນກາງເວັນ)</span>
              <strong>ປິດການຂາຍ & ກັ່ນຕອງທັກຜີ</strong>
            </div>
            <div class="card-content">
              <h4>ສິ່ງທີ່ຕ້ອງເຮັດ:</h4>
              <ul>
                <li><strong>Speed to Lead:</strong> ຕອບແຊັດໃຫ້ໄວພາຍໃນ 60 ວິນາທີ ເພາະລູກຄ້າເລື່ອນເບິ່ງຫຼາຍເພຈພ້ອມກັນ.</li>
                <li><strong>ກົດ 15 ນາທີແຍກທັກຜີ:</strong> ສົ່ງລາຄາສິນຄ້າ + ໂປຣໂມຊັ່ນສົ່ງຟຣີ. ຖ້າເຂົາອ່ານແລ້ວມິດເກີນ 15 ນາທີ ➔ <strong>Move on ທັນທີ</strong> ບໍ່ຕ້ອງນັ່ງຖ້າ.</li>
                <li><strong>ລູກຄ້າແທ້:</strong> ເມື່ອລູກຄ້າໃຫ້ເບີໂທ ➔ ກົດສ້າງອໍເດີໃນລະບົບທັນທີ ເພື່ອສົ່ງສັນຍານ CAPI ບອກ Facebook.</li>
              </ul>
              <div class="rule-box info">
                 <strong>Tip:</strong> ຖ້າພົບຄົນກວນເມືອງ ຫຼື ເດັກນ້ອຍຫຼິ້ນເກມ ➔ ກົດ <strong>Ban from Page</strong> ທັນທີ.
              </div>
            </div>
          </div>

          <!-- Card 3: Evening -->
          <div class="time-card evening">
            <div class="time-badge evening">
              <span> 20:30 - 21:30 (ຕອນຄ່ຳ)</span>
              <strong>ເວລາຕັດສິນໃຈ (Kill, Hold, Scale)</strong>
            </div>
            <div class="card-content">
              <h4>ສິ່ງທີ່ຕ້ອງເຮັດ:</h4>
              <ul>
                <li>ເປີດເບິ່ງໜ້າ <strong>Ads Battle</strong> ເລືອກວັນທີ "ມື້ນີ້ (Today)".</li>
                <li><strong> KILL:</strong> ແອດທີ່ໃຊ້ງົບ $6.00 - $8.00 ແລ້ວຍັງ 0 ອໍເດີ ➔ ກົດປິດທັນທີ!</li>
                <li><strong> HOLD:</strong> ແອດທີ່ຫາກໍ່ໃຊ້ງົບ $1 - $3 ➔ ປ່ອຍແລ່ນຕໍ່ໃຫ້ຄົບ 48 ຊົ່ວໂມງ.</li>
                <li><strong> SCALE:</strong> ແອດນາງຟ້າທີ່ມີ 2-3 ອໍເດີ ແລະ CPA ຖືກ ➔ ກົດ Scale ເພີ່ມງົບ +20%.</li>
              </ul>
              <div class="rule-box success">
                 <strong>ເຫດຜົນ:</strong> ການປັບງົບຕອນ 21:00 ຈະເຮັດໃຫ້ລະບົບ Meta Bidding ເລີ່ມຕົ້ນງົບໃໝ່ຢ່າງສະໝ່ຳສະເໝີໃນວັນຕໍ່ໄປ (00:00).
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- ========================================== -->
      <!-- TAB 2: ວິທີອ່ານ 4 ຕົວເລກ (Diagnostics) -->
      <!-- ========================================== -->
      <section v-if="activeTab === 'metrics'" class="tab-section">
        <div class="section-header">
          <h2> ວິທີອ່ານ 4 ຕົວເລກຫຼັກ: ຮູ້ທັນທີວ່າແອດຕິດບັນຫາບ່ອນໃດ!</h2>
          <p class="section-lead">
            ເວລາ CPA (ຕົ້ນທຶນຕໍ່ອໍເດີ) ແພງ, ຢ່າຟ້າວດ່ວນສະຫຼຸບວ່າ "ແອດບໍ່ດີ". ໃຫ້ໄລ່ກວດ 4 ຂັ້ນຕອນນີ້ຕາມລຳດັບ:
          </p>
        </div>

        <div class="funnel-chain">
          <div class="funnel-step">
            <span class="step-num">1</span>
            <strong>CPM (ຄ່າສະແດງຜົນ)</strong>
            <small>ຄ່າເຫັນ 1,000 ເທື່ອ</small>
          </div>
          <div class="funnel-arrow">➔</div>
          <div class="funnel-step">
            <span class="step-num">2</span>
            <strong>Link CTR (ອັດຕາການຄລິກ)</strong>
            <small>ຄົນຢຸດເບິ່ງ & ກົດ</small>
          </div>
          <div class="funnel-arrow">➔</div>
          <div class="funnel-step">
            <span class="step-num">3</span>
            <strong>Cost per Message</strong>
            <small>ຕົ້ນທຶນຕໍ່ຄົນທັກ</small>
          </div>
          <div class="funnel-arrow">➔</div>
          <div class="funnel-step">
            <span class="step-num">4</span>
            <strong>Closing Rate & CPA</strong>
            <small>ປິດການຂາຍ & ກຳໄລ</small>
          </div>
        </div>

        <div class="metrics-grid mt-4">
          <!-- Metric 1 -->
          <div class="metric-card">
            <div class="metric-top">
              <span class="metric-tag">ຂັ້ນທີ 1</span>
              <h4>1. CPM (Cost per 1,000 Impressions)</h4>
            </div>
            <p class="metric-desc">
              ບອກວ່າ <strong>"ກຸ່ມເປົ້າໝາຍແພງ ຫຼື ຖືກ"</strong> (ຄ່າສະແດງຜົນໃຫ້ຄົນເຫັນ 1,000 ເທື່ອ).
            </p>
            <div class="metric-benchmarks">
              <div class="bm-row green"><span> ດີຫຼາຍ:</span> <strong>$1.00 - $3.00 (22,000 - 66,000 ₭)</strong></div>
              <div class="bm-row red"><span> ແພງຜິດປົກກະຕິ:</span> <strong>$6.00 - $10.00+</strong></div>
            </div>
            <div class="metric-fix">
               <strong>ວິທີແກ້ຖ້າ CPM ແພງ:</strong> ປິດ Audience Network ທັນທີ, ຂະຫຍາຍອາຍຸເປັນ 23-48 ປີ, ຫຼື ຍິງແບບ Broad ບໍ່ຕ້ອງໃສ່ Interest.
            </div>
          </div>

          <!-- Metric 2 -->
          <div class="metric-card">
            <div class="metric-top">
              <span class="metric-tag">ຂັ້ນທີ 2</span>
              <h4>2. Link CTR (Click-Through Rate)</h4>
            </div>
            <p class="metric-desc">
              ບອກວ່າ <strong>"ຄລິບວິດີໂອ ຫຼື ຮູບພາບ ໜ້າສົນໃຈບໍ່"</strong> (ຄົນເຫັນ 100 ຄົນ ກົດຈັກຄົນ).
            </p>
            <div class="metric-benchmarks">
              <div class="bm-row green"><span> ດີຫຼາຍ:</span> <strong>2.0% - 4.0%+ (ແອດດຶງດູດສາຍຕາ)</strong></div>
              <div class="bm-row yellow"><span> ປານກາງ:</span> <strong>1.0% - 1.8%</strong></div>
              <div class="bm-row red"><span> ຄົນເລື່ອນຜ່ານ:</span> <strong>ຕ່ຳກວ່າ 0.9%</strong></div>
            </div>
            <div class="metric-fix">
               <strong>ວິທີແກ້ຖ້າ CTR ຕ່ຳ:</strong> ປ່ຽນ 3 ວິນາທີທຳອິດຂອງວິດີໂອ (Hook) ຫຼື ປ່ຽນຮູບໜ້າປົກ (Thumbnail) ໃໝ່ທັນທີ.
            </div>
          </div>

          <!-- Metric 3 -->
          <div class="metric-card">
            <div class="metric-top">
              <span class="metric-tag">ຂັ້ນທີ 3</span>
              <h4>3. Cost per Message (ຕົ້ນທຶນຕໍ່ຄົນທັກ)</h4>
            </div>
            <p class="metric-desc">
              ບອກວ່າ <strong>"ຄົນສົນໃຈຢາກຊື້ຫຼາຍໜ້ອຍປານໃດ"</strong> ສຳລັບສິນຄ້າ Smartwatch / Gadget.
            </p>
            <div class="metric-benchmarks">
              <div class="bm-row green"><span> ດີຫຼາຍ:</span> <strong>$0.20 - $0.50 (4,500 - 11,000 ₭/ທັກ)</strong></div>
              <div class="bm-row yellow"><span> ພໍຮັບໄດ້:</span> <strong>$0.60 - $0.90</strong></div>
              <div class="bm-row red"><span> ແພງເກີນໄປ:</span> <strong>ເກີນ $1.20 - $2.00+ ຕໍ່ທັກ</strong></div>
            </div>
            <div class="metric-fix">
               <strong>ວິທີແກ້ຖ້າຄ່າທັກແພງ:</strong> ປັບ Headline ຂໍ້ຄວາມເທິງແອດ, ເນັ້ນໂປຣໂມຊັ່ນສົ່ງຟຣີ, ແລະ ໃສ່ລາຄາຈະແຈ້ງ.
            </div>
          </div>

          <!-- Metric 4 -->
          <div class="metric-card">
            <div class="metric-top">
              <span class="metric-tag">ຂັ້ນທີ 4</span>
              <h4>4. Closing Rate & Real CPA</h4>
            </div>
            <p class="metric-desc">
              ບອກວ່າ <strong>"ທີມຂາຍປິດໄດ້ດີບໍ່ ແລະ ໄດ້ກຳໄລແທ້ບໍ່"</strong> (ຄົນທັກ 10 ຄົນ ປິດໄດ້ຈັກຄົນ).
            </p>
            <div class="metric-benchmarks">
              <div class="bm-row green"><span> ທີມຂາຍເກັ່ງ:</span> <strong>20% - 30% (CPA ຕ່ຳກວ່າກຳໄລ)</strong></div>
              <div class="bm-row red"><span> ຮົ່ວໄຫຼ (ອັນຕະລາຍ):</span> <strong>ຕ່ຳກວ່າ 10% (ທັກ 20 ຄົນ ຊື້ 1 ຄົນ)</strong></div>
            </div>
            <div class="metric-fix">
               <strong>ວິທີແກ້ຖ້າ Closing Rate ຕ່ຳ:</strong> ແອດບໍ່ໄດ້ຜິດ! ແຕ່ບົດຕອບແຊັດຊ້າ ຫຼື ບໍ່ສ້າງຄວາມໝັ້ນໃຈ ➔ ຕ້ອງເນັ້ນ *"ກວດເຄື່ອງກ່ອນຈ່າຍໄດ້"*.
            </div>
          </div>
        </div>
      </section>

      <!-- ========================================== -->
      <!-- TAB 3: ກົດເຫຼັກຕັດສິນໃຈ (Kill, Hold, Scale) -->
      <!-- ========================================== -->
      <section v-if="activeTab === 'decision'" class="tab-section">
        <div class="section-header">
          <h2> ກົດເຫຼັກການຕັດສິນໃຈ: 3 ຄຳສັ່ງ (Kill, Hold, Scale)</h2>
          <p class="section-lead">
            ທຸກໆຄ່ຳ (21:00), ໃຫ້ເປີດເບິ່ງໜ້າ Ads Battle ແລ້ວໃຊ້ກົດເຫຼັກນີ້ຕັດສິນທັນທີ ໂດຍບໍ່ໃຊ້ອາລົມ:
          </p>
        </div>

        <div class="decision-cards">
          <!-- KILL -->
          <div class="decision-card kill">
            <div class="decision-header">
              <span class="d-icon"></span>
              <div>
                <h3>1. ຄຳສັ່ງ KILL (ປິດແອດຖິ້ມທັນທີ)</h3>
                <small>ຢຸດງົບປະມານຮົ່ວໄຫຼ</small>
              </div>
            </div>
            <div class="decision-criteria">
              <h4>ເງື່ອນໄຂທີ່ຕ້ອງປິດ:</h4>
              <ul>
                <li>ແອດທີ່ໃຊ້ງົບໄປຮອດ <strong>$6.00 - $8.00+ ແລ້ວຍັງ 0 ອໍເດີ</strong> ➔ ປິດທັນທີ!</li>
                <li>ແອດທີ່ <strong>Link CTR ຕ່ຳກວ່າ 0.8%</strong> ຕິດຕໍ່ກັນ 2 ມື້ (ຄົນບໍ່ສົນໃຈ) ➔ ປິດທັນທີ!</li>
                <li>ແອດທີ່ <strong>Frequency (ຄວາມຖີ່) ສູງເກີນ 2.5x</strong> (ຄົນເຫັນຊ້ຳໆຈົນເບື່ອ) ➔ ປິດແລ້ວປ່ຽນຄລິບໃໝ່!</li>
              </ul>
            </div>
          </div>

          <!-- HOLD -->
          <div class="decision-card hold">
            <div class="decision-header">
              <span class="d-icon"></span>
              <div>
                <h3>2. ຄຳສັ່ງ HOLD (ປ່ອຍແລ່ນຕໍ່ ຫ້າມປິດ!)</h3>
                <small>ໃຫ້ເວລາ AI ຮຽນຮູ້ (Learning Phase)</small>
              </div>
            </div>
            <div class="decision-criteria">
              <h4>ເງື່ອນໄຂທີ່ຫ້າມປິດ:</h4>
              <ul>
                <li>ແອດຫາກໍ່ເປີດໄດ້ 24 ຊົ່ວໂມງ, ໃຊ້ງົບໄປພຽງ <strong>$1.00 - $3.00 ➔ ຫ້າມປິດເດັດຂາດ!</strong></li>
                <li>ຄົນທັກຫຼາຍ (Cost per message ຖືກ $0.25 - $0.40) ແຕ່ລູກຄ້າກຳລັງລົມ ຫຼື ລໍຖ້າເລີກວຽກມາໂອນ ➔ ປ່ອຍແລ່ນຕໍ່ອີກ 24 ຊົ່ວໂມງ.</li>
              </ul>
            </div>
          </div>

          <!-- SCALE -->
          <div class="decision-card scale">
            <div class="decision-header">
              <span class="d-icon"></span>
              <div>
                <h3>3. ຄຳສັ່ງ SCALE (ເພີ່ມງົບຂະຫຍາຍຍອດຂາຍ)</h3>
                <small>ດັນແອດນາງຟ້າໃຫ້ສ້າງກຳໄລສູງສຸດ</small>
              </div>
            </div>
            <div class="decision-criteria">
              <h4>ເງື່ອນໄຂ ແລະ ວິທີ Scale:</h4>
              <ul>
                <li>ແອດທີ່ສ້າງອໍເດີໄດ້ <strong>2-3 ອໍເດີຂຶ້ນໄປ</strong> ແລະ <strong>CPA ຕ່ຳກວ່າກຳໄລຕໍ່ຊິ້ນ</strong> (ເຊັ່ນ CPA $1.04 - $1.77).</li>
                <li> <strong>ວິທີ Scale ທີ່ຖືກຕ້ອງ:</strong> ກົດປຸ່ມ <strong>Scale +20%</strong> ໃນໜ້າ Ads Battle (ເຊັ່ນ $3.00 ➔ $3.60 ➔ $4.30).</li>
                <li> <strong>ຫ້າມເພີ່ມງົບກະໂດດ 100%</strong> (ເຊັ່ນ ຈາກ $3 ກະໂດດໄປ $10) ເພາະລະບົບຈະ Restart Learning Phase ແລະ ເຮັດໃຫ້ຄ່າແອດແພງຂຶ້ນທັນທີ!</li>
              </ul>
            </div>
          </div>
        </div>
      </section>

      <!-- ========================================== -->
      <!-- TAB 4: ວິທີຈັດການ "ທັກຜີ" & ມືຫຼົງກົດ -->
      <!-- ========================================== -->
      <section v-if="activeTab === 'ghost'" class="tab-section">
        <div class="section-header">
          <h2> ວິທີຈັດການ "ທັກຜີ" & ມືຫຼົງກົດ: ບໍ່ໃຫ້ມາແດກຄ່າແອດ!</h2>
          <p class="section-lead">
            ມືຫຼົງກົດເກີດຈາກຄົນຫຼິ້ນເກມມືຖືແລ້ວມືພາດໄປໂດນປຸ່ມ Message ຫຼື ເດັກນ້ອຍກົດຫຼິ້ນ. ນີ້ຄື 3 ວິທີປິດກັ້ນແບບເດັດຂາດ:
          </p>
        </div>

        <div class="ghost-steps-list">
          <!-- Step 1 -->
          <div class="ghost-step-card">
            <div class="g-num">1</div>
            <div class="g-body">
              <h4> ປິດ "Audience Network" ໃນ Ad Set ທັນທີ</h4>
              <p>
                ໃນ Ad Set ➔ Placements ➔ Manual Placements: <strong>ຕິກເອົາ Audience Network ອອກ!</strong> ເພາະເກືອບທັງໝົດຂອງທັກຜີ ແມ່ນຄົນຫຼິ້ນເກມມືຖືແລ້ວພະຍາຍາມກົດປິດໂຄສະນາແຕ່ມືໄປໂດນປຸ່ມ Send Message!
              </p>
            </div>
          </div>

          <!-- Step 2 -->
          <div class="ghost-step-card">
            <div class="g-num">2</div>
            <div class="g-body">
              <h4> ໃສ່ລາຄາສິນຄ້າຕົວໃຫຍ່ໆໃສ່ໜ້າປົກຮູບ ຫຼື ຄລິບແອດ</h4>
              <p>
                ຢ່າປິດບັງລາຄາ! ຄົນທີ່ທັກມາຖາມວ່າ <em>"ລາຄາເທົ່າໃດ?"</em> ແລ້ວມິດງຽບ ເພາະເຂົາຄິດວ່າລາຄາຖືກກວ່ານັ້ນ. ຖ້າໃສ່ລາຄາສິນຄ້າໂຊຈະແຈ້ງ, ຄົນທີ່ທັກມາຈະເປັນຄົນທີ່ <strong>"ພ້ອມຈ່າຍແລ້ວ"</strong> ເທົ່ານັ້ນ!
              </p>
            </div>
          </div>

          <!-- Step 3 -->
          <div class="ghost-step-card">
            <div class="g-num">3</div>
            <div class="g-body">
              <h4> ຕັ້ງ Exclude: ບໍ່ສົ່ງແອດໃຫ້ຄົນທີ່ເຄີຍທັກເພຈໃນ 30 ມື້</h4>
              <p>
                ໃນ Ad Set ➔ Audience ➔ ຊ່ອງ <strong>Exclude</strong>: ເລືອກ <code>People who sent a message in 30 days</code>.
                ພໍໃຜກົດທັກມາ 1 ເທື່ອ (ແມ້ແຕ່ມືຫຼົງກົດ), <strong>Facebook ຈະບລັອກບໍ່ໃຫ້ເຂົາເຫັນແອດອີກ 30 ມື້ທັນທີ!</strong>
              </p>
            </div>
          </div>

          <!-- Step 4 -->
          <div class="ghost-step-card">
            <div class="g-num">4</div>
            <div class="g-body">
              <h4> ບລັອກລາຍຄົນ (Ban from Page ສຳລັບພວກກວນເມືອງ)</h4>
              <p>
                ຖ້າພົບຄົນທີ່ກົດຫຼິ້ນກວນເມືອງ: ໃຫ້ກົດຈຸດ 3 ຈ້ຳໃນແຊັດ ➔ ເລືອກ <strong>"Ban from Page"</strong> ເຂົາຈະບໍ່ສາມາດເຫັນໂພສ ຫຼື ແອດຂອງເພຈໄດ້ອີກຕະຫຼອດໄປ.
              </p>
            </div>
          </div>
        </div>
      </section>


    </main>
  </div>
</template>

<style scoped>
.ads-playbook-page {
  padding: 16px 20px 48px;
  max-width: 1200px;
  margin: 0 auto;
}

/* Hero Header */
.playbook-hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 24px 28px;
  background: linear-gradient(135deg, var(--color-text) 0%, var(--color-text) 100%);
  border-radius: 16px;
  color: var(--color-surface);
  margin-bottom: 20px;
  box-shadow: 0 4px 14px rgba(15, 23, 42, 0.12);
}

.hero-eyebrow {
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.1em;
  color: #38bdf8;
  display: block;
  margin-bottom: 4px;
}

.hero-title {
  font-size: 24px;
  font-weight: 850;
  margin: 0 0 6px 0;
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.hero-badge {
  font-size: 12px;
  background: rgba(56, 189, 248, 0.15);
  border: 1px solid rgba(56, 189, 248, 0.3);
  color: #38bdf8;
  padding: 2px 8px;
  border-radius: 999px;
  font-weight: 700;
}

.hero-sub {
  font-size: 13.5px;
  color: var(--color-text-faint);
  margin: 0;
  line-height: 1.5;
}

.hero-actions {
  display: flex;
  gap: 10px;
  flex-shrink: 0;
}

/* Nav Bar */
.playbook-nav {
  display: flex;
  gap: 8px;
  padding: 6px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  margin-bottom: 24px;
  overflow-x: auto;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.nav-tab-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  font-size: 13px;
  font-weight: 700;
  color: var(--color-text-muted);
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.15s;
}

.nav-tab-btn:hover {
  background: var(--color-background);
  color: var(--color-text);
}

.nav-tab-btn.active {
  background: var(--color-primary-soft);
  color: var(--color-primary);
  border-color: var(--color-info-border);
}

.tab-icon {
  font-size: 16px;
}

/* Body & Sections */
.tab-section {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.section-header h2 {
  font-size: 20px;
  font-weight: 800;
  color: var(--color-text);
  margin: 0 0 4px 0;
}

.section-lead {
  font-size: 13.5px;
  color: var(--color-text-muted);
  margin: 0;
  line-height: 1.6;
}

/* Time Cards */
.time-cards-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 16px;
}

.time-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 14px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.time-badge {
  padding: 12px 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12.5px;
  font-weight: 750;
}

.time-badge.morning { background: var(--color-primary-soft); color: #1e40af; border-bottom: 2px solid var(--color-primary); }
.time-badge.afternoon { background: var(--color-warning-soft); color: #9a3412; border-bottom: 2px solid #f97316; }
.time-badge.evening { background: var(--color-purple-soft); color: #6b21a8; border-bottom: 2px solid #a855f7; }

.card-content {
  padding: 16px;
}

.card-content h4 {
  font-size: 13px;
  font-weight: 750;
  color: var(--color-text);
  margin: 0 0 8px 0;
}

.card-content ul {
  margin: 0 0 14px 0;
  padding-left: 20px;
  font-size: 13px;
  line-height: 1.6;
  color: #334155;
}

.card-content li {
  margin-bottom: 6px;
}

.rule-box {
  padding: 10px 12px;
  border-radius: 8px;
  font-size: 12.5px;
  line-height: 1.5;
}

.rule-box.warn { background: var(--color-error-soft); border: 1px solid var(--color-error-border); color: var(--color-error); }
.rule-box.info { background: var(--color-success-soft); border: 1px solid var(--color-success-border); color: var(--color-success); }
.rule-box.success { background: #fdf4ff; border: 1px solid #f5d0fe; color: #86198f; }

/* Funnel Chain */
.funnel-chain {
  display: flex;
  align-items: center;
  gap: 10px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  padding: 14px 18px;
  overflow-x: auto;
}

.funnel-step {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  flex: 1;
  min-width: 140px;
}

.step-num {
  width: 24px;
  height: 24px;
  border-radius: 999px;
  background: var(--color-primary);
  color: var(--color-surface);
  font-size: 12px;
  font-weight: 800;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 4px;
}

.funnel-step strong {
  font-size: 13px;
  color: var(--color-text);
}

.funnel-step small {
  font-size: 12px;
  color: var(--color-text-muted);
}

.funnel-arrow {
  font-size: 18px;
  color: var(--color-text-faint);
}

/* Metrics Grid */
.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(270px, 1fr));
  gap: 16px;
}

.metric-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.metric-top {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.metric-tag {
  font-size: 12px;
  font-weight: 800;
  text-transform: uppercase;
  color: var(--color-primary);
  background: var(--color-primary-soft);
  padding: 1px 6px;
  border-radius: 4px;
  align-self: flex-start;
}

.metric-top h4 {
  font-size: 14.5px;
  font-weight: 800;
  color: var(--color-text);
  margin: 0;
}

.metric-desc {
  font-size: 12.5px;
  color: var(--color-text-secondary);
  margin: 0;
  line-height: 1.5;
}

.metric-benchmarks {
  display: flex;
  flex-direction: column;
  gap: 6px;
  background: var(--color-background);
  border: 1px solid var(--color-surface-soft);
  border-radius: 8px;
  padding: 10px 12px;
}

.bm-row {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
}

.bm-row.green strong { color: var(--color-success); }
.bm-row.yellow strong { color: var(--color-warning); }
.bm-row.red strong { color: var(--color-error); }

.metric-fix {
  font-size: 12px;
  color: var(--color-text);
  background: var(--color-surface-soft);
  padding: 8px 10px;
  border-radius: 6px;
  line-height: 1.5;
}

/* Decision Cards */
.decision-cards {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.decision-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 14px;
  padding: 18px 22px;
}

.decision-card.kill {
  border-color: rgba(239, 68, 68, 0.35);
  background: linear-gradient(180deg, rgba(239, 68, 68, 0.03) 0%, var(--color-surface) 100%);
}
.decision-card.hold {
  border-color: rgba(245, 158, 11, 0.35);
  background: linear-gradient(180deg, rgba(245, 158, 11, 0.03) 0%, var(--color-surface) 100%);
}
.decision-card.scale {
  border-color: rgba(16, 185, 129, 0.35);
  background: linear-gradient(180deg, rgba(16, 185, 129, 0.03) 0%, var(--color-surface) 100%);
}

.decision-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.d-icon { font-size: 26px; }

.decision-header h3 {
  font-size: 16px;
  font-weight: 800;
  color: var(--color-text);
  margin: 0;
}

.decision-header small {
  font-size: 12px;
  color: var(--color-text-muted);
}

.decision-criteria h4 {
  font-size: 13px;
  font-weight: 750;
  color: var(--color-text);
  margin: 0 0 6px 0;
}

.decision-criteria ul {
  margin: 0;
  padding-left: 20px;
  font-size: 13px;
  line-height: 1.6;
  color: #334155;
}

.decision-criteria li {
  margin-bottom: 4px;
}

/* Ghost Steps */
.ghost-steps-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.ghost-step-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  padding: 16px;
  display: flex;
  align-items: flex-start;
  gap: 14px;
}

.g-num {
  width: 32px;
  height: 32px;
  border-radius: 10px;
  background: var(--color-primary-soft);
  color: var(--color-primary);
  font-size: 15px;
  font-weight: 800;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.g-body h4 {
  font-size: 14.5px;
  font-weight: 800;
  color: var(--color-text);
  margin: 0 0 4px 0;
}

.g-body p {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin: 0;
  line-height: 1.6;
}



@media (max-width: 768px) {
  .playbook-hero {
    flex-direction: column;
    align-items: flex-start;
  }
  .hero-actions {
    width: 100%;
  }
  .funnel-chain {
    flex-direction: column;
    align-items: stretch;
  }
  .funnel-arrow {
    transform: rotate(90deg);
    text-align: center;
  }
}
</style>
