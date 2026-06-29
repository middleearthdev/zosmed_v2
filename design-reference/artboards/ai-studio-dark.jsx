function AIStudioDark() {
  const W = 1440, H = 1100;
  return (
    <div style={{ width: W, height: H, background: '#0a0a0a', color: '#f4f4f0', fontFamily: 'var(--font-sans)', display: 'flex' }}>
      <Sidebar active="ai"/>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        <div style={{ height: 56, borderBottom: '1px solid #17171a', display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '0 24px' }}>
          <span style={{ fontSize: 14, fontWeight: 500 }}>AI Studio</span>
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            <span className="mono" style={{ fontSize: 11, color: '#66665f' }}>last trained 12 min ago · 187 conversations</span>
            <button className="btn-ghost" style={{ padding: '7px 12px', fontSize: 12 }}>Test playground</button>
            <button className="btn-lime" style={{ padding: '7px 12px', fontSize: 12 }}><I.sparkle/> Retrain</button>
          </div>
        </div>
        <div style={{ flex: 1, overflow: 'auto', padding: 24, display: 'grid', gridTemplateColumns: '1.3fr 1fr', gap: 14 }}>
          {/* LEFT */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
            <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 22 }}>
              <div className="mono tracked" style={{ fontSize: 9.5, color: '#66665f', marginBottom: 14 }}>BRAND VOICE</div>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
                {[
                  { l: 'Formality', v: 0.32, lo: 'casual', hi: 'formal' },
                  { l: 'Warmth', v: 0.78, lo: 'cool', hi: 'warm' },
                  { l: 'Energy', v: 0.65, lo: 'calm', hi: 'energetic' },
                  { l: 'Humor', v: 0.45, lo: 'serious', hi: 'playful' },
                ].map(s => (
                  <div key={s.l}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12.5, marginBottom: 6 }}>
                      <span>{s.l}</span>
                      <span className="mono" style={{ color: ZZ_LIME }}>{Math.round(s.v * 100)}%</span>
                    </div>
                    <div style={{ position: 'relative', height: 6, background: '#17171a', borderRadius: 3 }}>
                      <div style={{ width: `${s.v * 100}%`, height: 6, background: ZZ_LIME, borderRadius: 3 }}/>
                      <span style={{ position: 'absolute', left: `calc(${s.v * 100}% - 6px)`, top: -3, width: 12, height: 12, borderRadius: 999, background: ZZ_LIME, border: '2px solid #0a0a0a' }}/>
                    </div>
                    <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 10.5, color: '#66665f', marginTop: 6 }} className="mono">
                      <span>{s.lo}</span><span>{s.hi}</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 22 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
                <span className="mono tracked" style={{ fontSize: 9.5, color: '#66665f' }}>SYSTEM PROMPT</span>
                <Pill tone="lime">v3.2</Pill>
              </div>
              <div style={{ background: '#0a0a0a', border: '1px solid #1a1a1d', borderRadius: 8, padding: 14, fontFamily: 'var(--font-mono)', fontSize: 12.5, lineHeight: 1.6, color: '#a3a39c' }}>
                <span style={{ color: ZZ_LIME }}>Kamu</span> adalah asisten DM untuk <span style={{ color: 'oklch(0.78 0.16 240)' }}>@ataka.studio</span>,<br/>
                brand fashion sustainable dari Bandung.<br/><br/>
                <span style={{ color: '#66665f' }}>// tone</span><br/>
                Friendly, casual, sedikit playful. Pakai emoji secukupnya 💚.<br/>
                Bahasa Indonesia santai (kak/sis), boleh switch ke English<br/>
                kalau user mulai duluan.<br/><br/>
                <span style={{ color: '#66665f' }}>// rules</span><br/>
                – Jangan buat janji harga sebelum cek <span style={{ background: 'oklch(0.85 0.16 75 / 0.18)', color: 'oklch(0.85 0.16 75)', padding: '0 4px', borderRadius: 3 }}>{'{{products.csv}}'}</span><br/>
                – Hand-off ke admin kalau topic = refund/komplain<br/>
                – Selalu close dengan CTA (link/order/follow)
              </div>
            </div>

            <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 22 }}>
              <div className="mono tracked" style={{ fontSize: 9.5, color: '#66665f', marginBottom: 14 }}>KNOWLEDGE BASE · 4 SOURCES</div>
              {[
                { f: 'products.csv', m: '128 SKU · 1.4 MB', last: '2h ago', status: 'synced' },
                { f: 'faq.md', m: '47 Q&A · 8 KB', last: '1d ago', status: 'synced' },
                { f: 'shipping-policy.pdf', m: '12 pages · 240 KB', last: '3d ago', status: 'synced' },
                { f: 'instagram-bio.txt', m: '420 chars', last: '1w ago', status: 'stale' },
              ].map((f, i) => (
                <div key={f.f} style={{ display: 'grid', gridTemplateColumns: '24px 1fr 120px 90px 60px', gap: 12, alignItems: 'center', padding: '10px 0', borderTop: i ? '1px solid #1a1a1d' : 'none' }}>
                  <span style={{ width: 20, height: 20, background: '#17171a', borderRadius: 4, display: 'inline-flex', alignItems: 'center', justifyContent: 'center', color: '#a3a39c' }} className="mono">📄</span>
                  <span className="mono" style={{ fontSize: 12.5 }}>{f.f}</span>
                  <span className="mono" style={{ fontSize: 11, color: '#66665f' }}>{f.m}</span>
                  <span className="mono" style={{ fontSize: 11, color: '#66665f' }}>{f.last}</span>
                  <Pill tone={f.status === 'synced' ? 'lime' : 'warn'}>{f.status}</Pill>
                </div>
              ))}
              <div style={{ marginTop: 12, padding: 12, border: '1px dashed #2a2a2e', borderRadius: 8, textAlign: 'center', color: '#66665f', fontSize: 12 }} className="mono">
                + drop file or connect data source (Notion · Sheets · Webhook)
              </div>
            </div>
          </div>

          {/* RIGHT */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
            <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 22 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
                <span className="mono tracked" style={{ fontSize: 9.5, color: '#66665f' }}>TEST PLAYGROUND</span>
                <Pill tone="blue">model: gpt-4o-mini</Pill>
              </div>
              <div style={{ background: '#0a0a0a', border: '1px solid #1a1a1d', borderRadius: 8, padding: 14, display: 'flex', flexDirection: 'column', gap: 8 }}>
                <ChatBubble side="them" text='"berapa harga yang sage size M? bisa cod jaksel ga?"'/>
                <ChatBubble side="us" ai text="Hai kak! Yang sage size M Rp 189rb 💚 COD jaksel BISA banget — minimum order Rp 150rb, free dalam ring 1. Mau aku kirim link order?"/>
                <div style={{ display: 'flex', gap: 6, marginTop: 4 }}>
                  <span className="mono" style={{ fontSize: 10, padding: '3px 7px', background: 'oklch(0.78 0.16 240 / 0.18)', color: 'oklch(0.78 0.16 240)', borderRadius: 3 }}>used: products.csv</span>
                  <span className="mono" style={{ fontSize: 10, padding: '3px 7px', background: 'oklch(0.85 0.16 75 / 0.18)', color: 'oklch(0.85 0.16 75)', borderRadius: 3 }}>used: shipping-policy.pdf</span>
                  <span className="mono" style={{ fontSize: 10, padding: '3px 7px', background: 'oklch(0.9 0.2 130 / 0.18)', color: ZZ_LIME, borderRadius: 3, marginLeft: 'auto' }}>1.2s · 248 tok</span>
                </div>
              </div>
              <div style={{ marginTop: 12, padding: 10, background: '#0a0a0a', border: '1px solid #1a1a1d', borderRadius: 8, display: 'flex', alignItems: 'center', gap: 8 }}>
                <span style={{ fontSize: 13, color: '#66665f', flex: 1 }}>type your test message…</span>
                <button className="btn-lime" style={{ padding: '6px 12px', fontSize: 12 }}>Run <I.send/></button>
              </div>
            </div>

            <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 22 }}>
              <div className="mono tracked" style={{ fontSize: 9.5, color: '#66665f', marginBottom: 14 }}>EVAL · LAST 7 DAYS</div>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 12 }}>
                {[
                  ['Helpfulness', '92%', '+3pt'],
                  ['Tone match', '88%', '+5pt'],
                  ['CTA included', '76%', '−2pt'],
                  ['Hand-off rate', '4.2%', '−1.8pt'],
                ].map(([k, v, d]) => (
                  <div key={k} style={{ background: '#0a0a0a', border: '1px solid #1a1a1d', borderRadius: 8, padding: 14 }}>
                    <div className="mono tracked" style={{ fontSize: 9.5, color: '#66665f' }}>{k}</div>
                    <div className="mono" style={{ fontSize: 24, fontWeight: 500, marginTop: 4 }}>{v}</div>
                    <div className="mono" style={{ fontSize: 11, color: d.startsWith('+') ? ZZ_LIME : 'oklch(0.78 0.2 0)', marginTop: 2 }}>{d}</div>
                  </div>
                ))}
              </div>
            </div>

            <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 22 }}>
              <div className="mono tracked" style={{ fontSize: 9.5, color: '#66665f', marginBottom: 14 }}>RECENT HAND-OFFS · NEED REVIEW</div>
              {[
                ['budi.s', 'asked refund policy', 'low confidence (0.42)'],
                ['nadya.p', 'complaint about shipping', 'sentiment: negative'],
                ['putu.gita', 'asked custom order', 'no kb match'],
              ].map(([u, q, r]) => (
                <div key={u} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 0', borderTop: '1px solid #1a1a1d' }}>
                  <Avatar name={u.slice(0,2).toUpperCase()} color="#3a3a40" size={28}/>
                  <div style={{ flex: 1 }}>
                    <div className="mono" style={{ fontSize: 12 }}>@{u}</div>
                    <div style={{ fontSize: 12, color: '#a3a39c' }}>{q} · <span className="mono" style={{ fontSize: 10.5, color: 'oklch(0.85 0.16 75)' }}>{r}</span></div>
                  </div>
                  <span className="mono" style={{ fontSize: 11, color: ZZ_LIME }}>Review →</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}