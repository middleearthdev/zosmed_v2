'use client';

/**
 * Right-hand config panel (ADR-004 F5). Renders a dedicated form for the
 * two configurable runnable node kinds (`keyword-match`, `send-whatsapp-link`)
 * and a plain info panel for everything else — config always flows back
 * into canvas state via `onChangeConfig` (never fetched mid-JSX, §12a-3).
 */
import type { ReactNode } from 'react';
import { I, type IconName } from '@zosmed/ui';
import type { KeywordMatchConfig, NodeCatalogEntry, SendWhatsappLinkConfig, WorkflowNode } from '@zosmed/types';
import { iconForNodeType, wfTypeForCategory } from '@/lib/workflow-catalog';
import { NODE_COLORS } from '@/lib/mock/workflows';

export function NodeInspector({
  node,
  catalogEntry,
  onChangeConfig,
  onRemove,
}: {
  node: WorkflowNode | null;
  catalogEntry: NodeCatalogEntry | undefined;
  onChangeConfig: (config: Record<string, unknown>) => void;
  onRemove: () => void;
}) {
  if (!node) {
    return (
      <div className="border-bg-3 flex w-[320px] flex-col items-center justify-center gap-2 border-l px-6 py-10 text-center">
        <span className="text-text-3 text-xs">Pilih node di canvas untuk mengatur konfigurasinya.</span>
        <span className="text-text-3 mono text-[10.5px]">Atau klik salah satu node di palette untuk menambah node baru.</span>
      </div>
    );
  }

  const color = NODE_COLORS[wfTypeForCategory(node.node.category)];
  const iconKey: IconName = iconForNodeType(node.node.kind);

  return (
    <div className="border-bg-3 w-[320px] overflow-y-auto border-l px-[18px] py-5">
      <div className="mb-1 flex items-center gap-2">
        <span
          className="inline-flex h-7 w-7 items-center justify-center rounded-md"
          style={{ background: `color-mix(in oklch, ${color} 14%, transparent)`, color }}
        >
          {I[iconKey]()}
        </span>
        <span className="mono tracked text-[10px]" style={{ color }}>
          {node.node.category.toUpperCase()} · {node.node.kind}
        </span>
      </div>
      <h3 className="mb-1 mt-1.5 text-lg font-medium">{node.label}</h3>
      <p className="text-text-3 m-0 text-xs">{catalogEntry?.description ?? 'Node dari katalog feasible (CLAUDE.md §7).'}</p>

      <div className="bg-bg-3 my-5 h-px" />

      {!catalogEntry?.runnable ? (
        <div
          className="mb-[18px] flex items-start gap-2 rounded-[7px] px-2.5 py-2"
          style={{ background: 'oklch(0.85 0.16 75 / 0.1)', border: '1px solid oklch(0.85 0.16 75 / 0.3)' }}
        >
          <span className="mt-px text-xs" style={{ color: 'var(--zz-warn)' }}>
            ⓘ
          </span>
          <span className="text-text-2 text-[11px] leading-normal">
            Node ini belum bisa dijalankan otomatis (segera hadir). Workflow tidak bisa dipublish kalau node ini jadi satu-satunya
            trigger/action.
          </span>
        </div>
      ) : null}

      {node.node.kind === 'keyword-match' ? (
        <KeywordMatchFields config={node.config as unknown as KeywordMatchConfig} onChange={onChangeConfig} />
      ) : node.node.kind === 'send-whatsapp-link' ? (
        <SendWhatsappLinkFields config={node.config as unknown as SendWhatsappLinkConfig} onChange={onChangeConfig} />
      ) : (
        <Field label="KONFIGURASI">
          <div className="bg-bg-2 border-line rounded-lg border p-3 text-[12.5px] leading-normal text-text-2">
            {catalogEntry?.runnable
              ? 'Node ini tidak butuh konfigurasi tambahan — siap dipakai langsung di workflow.'
              : 'Konfigurasi untuk node ini menyusul di iterasi berikutnya.'}
          </div>
        </Field>
      )}

      <div className="bg-bg-3 my-5 h-px" />

      <button
        type="button"
        onClick={onRemove}
        className="mono w-full rounded-md border px-3 py-2 text-center text-xs transition-colors"
        style={{ borderColor: 'oklch(0.78 0.2 0 / 0.35)', color: 'var(--zz-pink)' }}
      >
        Hapus node
      </button>
    </div>
  );
}

function KeywordMatchFields({
  config,
  onChange,
}: {
  config: KeywordMatchConfig;
  onChange: (config: Record<string, unknown>) => void;
}) {
  const keywords = Array.isArray(config?.keywords) ? config.keywords : [];
  return (
    <>
      <Field label="KATA KUNCI (pisahkan dengan koma)">
        <textarea
          defaultValue={keywords.join(', ')}
          onBlur={(e) => {
            const next = e.target.value
              .split(',')
              .map((k) => k.trim())
              .filter(Boolean);
            onChange({ keywords: next });
          }}
          rows={3}
          placeholder="keep, c1, c3, order"
          className="bg-bg-2 border-line text-text placeholder:text-text-3 w-full rounded-lg border p-3 text-[13px] leading-normal outline-none focus:border-lime"
        />
      </Field>
      <Field label="MODE PENCOCOKAN">
        <label className="bg-bg-2 border-line flex items-center gap-2 rounded-md border px-2.5 py-2 text-xs">
          <input
            type="checkbox"
            defaultChecked={config?.caseInsensitive ?? true}
            onChange={(e) => onChange({ caseInsensitive: e.target.checked })}
          />
          Abaikan besar/kecil huruf (case insensitive)
        </label>
      </Field>
    </>
  );
}

function SendWhatsappLinkFields({
  config,
  onChange,
}: {
  config: SendWhatsappLinkConfig;
  onChange: (config: Record<string, unknown>) => void;
}) {
  return (
    <>
      <Field label="TEMPLATE PESAN">
        <textarea
          defaultValue={config?.template ?? ''}
          onBlur={(e) => onChange({ template: e.target.value })}
          rows={5}
          placeholder="Halo {{nama}}, makasih udah komen di {{post}}! Yuk lanjut chat di WhatsApp ya 💬"
          className="bg-bg-2 border-line text-text placeholder:text-text-3 w-full rounded-lg border p-3 text-[13px] leading-normal outline-none focus:border-lime"
        />
        <span className="mono text-text-3 mt-1.5 block text-[10.5px]">
          Variabel: {'{{nama}}'} · {'{{produk}}'} · {'{{post}}'}
        </span>
      </Field>
      <Field label="NOMOR WHATSAPP TUJUAN">
        <input
          type="text"
          defaultValue={config?.waPhone ?? ''}
          onBlur={(e) => onChange({ waPhone: e.target.value })}
          placeholder="6281234567890"
          className="bg-bg-2 border-line text-text placeholder:text-text-3 w-full rounded-lg border px-3 py-2.5 text-sm outline-none focus:border-lime"
        />
        <span className="mono text-text-3 mt-1.5 block text-[10.5px]">Format internasional, tanpa tanda &quot;+&quot;.</span>
      </Field>
    </>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="mb-[18px]">
      <div className="mono tracked text-text-3 mb-2 text-[9.5px]">{label}</div>
      {children}
    </div>
  );
}
