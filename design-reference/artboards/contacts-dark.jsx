// ─────────── CONTACTS ─────────── (from menu-details-dark.jsx)
function ContactsDark() {
  const W = 1440, H = 1100;
  const rows = [
    { u: 'rina_susanti', n: 'Rina Susanti', tags: ['warm-lead','jakarta'], ig: 2400, conv: 12, ltv: 'Rp 0', last: '2m', src: 'promo-launch-mei' },
    { u: 'arief.daud',   n: 'Arief Daud',   tags: ['hot-lead','bandung'], ig: 8800, conv: 21, ltv: 'Rp 540rb', last: '14m', src: 'promo-launch-mei' },
    { u: 'mira.hidayah', n: 'Mira Hidayah', tags: ['giveaway-entry'],     ig: 1200, conv: 4, ltv: 'Rp 0', last: '32m', src: 'giveaway-buku' },
    { u: 'budi.s',       n: 'Budi Saputra', tags: ['warm-lead','jakarta'], ig: 4100, conv: 18, ltv: 'Rp 189rb', last: '1h', src: 'faq-bot' },
    { u: 'nadya.p',      n: 'Nadya Putri',  tags: ['repeat-buyer'],       ig: 320,  conv: 47, ltv: 'Rp 1.8jt', last: '2h', src: 'organic' },
    { u: 'sintia.f',     n: 'Sintia F.',    tags: ['warm-lead'],          ig: 760,  conv: 8, ltv: 'Rp 0', last: '3h', src: 'promo-launch-mei' },
    { u: 'rizky_p',      n: 'Rizky P.',     tags: ['supporter'],          ig: 12800,conv: 3, ltv: 'Rp 0', last: '5h', src: 'organic' },
    { u: 'putu.gita',    n: 'Putu Gita',    tags: ['warm-lead','bali'],   ig: 540,  conv: 9, ltv: 'Rp 0', last: '1d', src: 'lead-magnet' },
    { u: 'lely.r',       n: 'Lely Rahma',   tags: ['repeat-buyer','vip'], ig: 1840, conv: 62, ltv: 'Rp 4.2jt', last: '1d', src: 'organic' },
    { u: 'dimas.o',      n: 'Dimas O.',     tags: ['cold'],               ig: 280,  conv: 1, ltv: 'Rp 0', last: '3d', src: 'faq-bot' },
  ];
  return (
    <div style={{ width: W, height: H, background: '#0a0a0a', color: '#f4f4f0', fontFamily: 'var(--font-sans)', display: 'flex' }}>
      <Sidebar active="contacts"/>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        {/* page-title header bar: "Contacts" + Export CSV (ghost) + New segment (lime, I.plus) */}
        <div style={{ height: 56, borderBottom: '1px solid #17171a', display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '0 24px' }}>
          <span style={{ fontSize: 14, fontWeight: 500 }}>Contacts</span>
          <div style={{ display: 'flex', gap: 8 }}>
            <button className="btn-ghost" style={{ padding: '7px 12px', fontSize: 12 }}>Export CSV</button>
            <button className="btn-lime" style={{ padding: '7px 12px', fontSize: 12 }}><I.plus/> New segment</button>
          </div>
        </div>
        <div style={{ flex: 1, overflow: 'auto', padding: 24 }}>
          {/* eyebrow + h1 */}
          <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between', marginBottom: 18 }}>
            <div>
              <span className="mono tracked" style={{ fontSize: 10, color: '#66665f' }}>14,287 CONTACTS · 247 NEW THIS WEEK</span>
              <h1 style={{ fontSize: 32, fontWeight: 500, letterSpacing: '-0.02em', margin: '6px 0 0' }}>People &amp; segments</h1>
            </div>
          </div>

          {/* segment chips — first active (lime bg dark text); each has label + count (count opacity 0.6) */}
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 18 }}>
            {[
              { l: 'All contacts', n: 14287, active: true },
              { l: 'Hot leads', n: 642 },
              { l: 'Warm leads', n: 2341 },
              { l: 'Repeat buyers', n: 412 },
              { l: 'VIP', n: 38 },
              { l: 'Cold', n: 8214 },
              { l: 'Giveaway entries', n: 1247 },
              { l: 'Jakarta', n: 4128 },
              { l: '+ new segment', n: null },
            ].map((s, i) => (
              <span key={i} className="mono" style={{
                padding: '6px 12px', fontSize: 12,
                background: s.active ? ZZ_LIME : '#111110',
                color: s.active ? '#0a0a0a' : '#a3a39c',
                border: s.active ? 'none' : '1px solid #232326',
                borderRadius: 999, display: 'inline-flex', gap: 8, alignItems: 'center',
              }}>
                {s.l}
                {s.n != null && <span style={{ opacity: 0.6 }}>{s.n.toLocaleString()}</span>}
              </span>
            ))}
          </div>

          {/* search + filters row */}
          <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
            <div style={{ flex: 1, display: 'flex', alignItems: 'center', gap: 8, padding: '8px 12px', background: '#111110', border: '1px solid #232326', borderRadius: 8 }}>
              <I.search/><span style={{ fontSize: 13, color: '#66665f' }}>Search by username, name, or tag…</span>
            </div>
            <button className="btn-ghost" style={{ padding: '8px 12px', fontSize: 12 }}><I.filter/> Filters · 0</button>
            <button className="btn-ghost" style={{ padding: '8px 12px', fontSize: 12 }}>Sort · Last seen ↓</button>
          </div>

          {/* table: grid cols 1.6fr 1.4fr 0.7fr 0.7fr 0.9fr 1fr 0.6fr; header bg #0d0d0d */}
          <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, overflow: 'hidden' }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1.6fr 1.4fr 0.7fr 0.7fr 0.9fr 1fr 0.6fr', padding: '10px 16px', borderBottom: '1px solid #232326', background: '#0d0d0d' }}>
              {['CONTACT', 'TAGS', 'IG FOLLOWERS', 'CONV.', 'LTV', 'SOURCE', 'LAST'].map(h => (
                <span key={h} className="mono tracked" style={{ fontSize: 9.5, color: '#66665f' }}>{h}</span>
              ))}
            </div>
            {rows.map((r, i) => (
              <div key={r.u} style={{ display: 'grid', gridTemplateColumns: '1.6fr 1.4fr 0.7fr 0.7fr 0.9fr 1fr 0.6fr', padding: '12px 16px', borderBottom: i < rows.length - 1 ? '1px solid #17171a' : 'none', alignItems: 'center', gap: 8 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                  <Avatar name={r.u.slice(0, 2).toUpperCase()} color="#3a3a40" size={28}/>
                  <div>
                    <div style={{ fontSize: 13, fontWeight: 500 }}>{r.n}</div>
                    <div className="mono" style={{ fontSize: 11, color: '#66665f' }}>@{r.u}</div>
                  </div>
                </div>
                <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
                  {/* Pill tone: hot→pink, vip→lime, else neutral */}
                  {r.tags.map(t => <Pill key={t} tone={t.includes('hot') ? 'pink' : t.includes('vip') ? 'lime' : 'neutral'}>{t}</Pill>)}
                </div>
                <span className="mono tnum" style={{ fontSize: 12 }}>{r.ig.toLocaleString()}</span>
                <span className="mono tnum" style={{ fontSize: 12 }}>{r.conv}</span>
                <span className="mono tnum" style={{ fontSize: 12, color: r.ltv === 'Rp 0' ? '#66665f' : ZZ_LIME }}>{r.ltv}</span>
                <span className="mono" style={{ fontSize: 11, color: '#a3a39c' }}>{r.src}</span>
                <span className="mono" style={{ fontSize: 11, color: '#66665f' }}>{r.last}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
// NOTE: Contact rows are clickable → navigate to /contacts/[u] (ContactProfileDark).
