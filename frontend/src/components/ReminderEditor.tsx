import { useState } from 'react';
import { FieldError, Modal } from './ui';
import { reminderLabel } from '../reminders';

export default function ReminderEditor({ label, offsets, busy, onSave }: { label: string; offsets: number[]; busy: boolean; onSave: (v: number[]) => Promise<void> }) {
 const [editing, setEditing] = useState<number | null>(null);
 const [amount, setAmount] = useState('15');
 const [unit, setUnit] = useState('1');
 const [error, setError] = useState('');
 const open = (index: number) => { setEditing(index); setAmount(String(offsets[index] ?? 15)); setUnit('1'); setError(''); };
 async function save() {
  const minutes = Number(amount) * Number(unit);
  if (!amount.trim() || !Number.isInteger(minutes) || minutes < 0 || minutes > 525600) { setError('Укажите время от 0 минут до 365 дней'); return; }
  if (offsets.some((v,i) => i !== editing && v === minutes)) { setError('Такое напоминание уже есть'); return; }
  const next = [...offsets]; next[editing!] = minutes;
  try { await onSave(next.sort((a,b) => b-a)); setEditing(null); } catch { setError('Сохранение не удалось. Попробуйте ещё раз.'); }
 }
 return <section className="reminder-editor" aria-label={`Напоминания: ${label}`}>
  <div className="row-head"><h3>{label}</h3><span className="settings-hint">{offsets.length}/5</span></div>
  <div className="reminder-list">{offsets.map((m,i) => <div className="reminder-row" key={m}>
   <button className="reminder-time" disabled={busy} onClick={() => open(i)}>{m === 0 && label === 'События' ? 'В момент начала' : reminderLabel(m)}</button>
   <button className="icon-btn remove-reminder" aria-label={`Удалить напоминание ${reminderLabel(m)}`} disabled={busy} onClick={() => void onSave(offsets.filter((_,j) => i!==j)).catch(() => {})}>×</button>
  </div>)}</div>
  {offsets.length < 5 && <button className="add-reminder" disabled={busy} onClick={() => open(offsets.length)}><span aria-hidden="true">+</span> Добавить напоминание</button>}
  {offsets.length === 0 && <p className="settings-hint">Напоминаний до срока нет.</p>}
  {editing !== null && <Modal title={`${label}: напоминание`} onClose={() => { if (!busy) setEditing(null); }}>
   <label className="field"><span>За сколько до {label === 'События' ? 'начала' : 'дедлайна'}</span><input type="number" min="0" step="1" value={amount} onChange={e=>setAmount(e.target.value)} /></label>
   <label className="field"><span>Единицы времени</span><select value={unit} onChange={e=>setUnit(e.target.value)}><option value="1">Минуты</option><option value="60">Часы</option><option value="1440">Дни</option></select></label>
   <p className="settings-hint">0 — уведомление в момент {label === 'События' ? 'начала события' : 'дедлайна'}.</p>
   <FieldError text={error} /><div className="modal-actions"><button className="btn-secondary" disabled={busy} onClick={()=>setEditing(null)}>Отмена</button><button className="btn-primary" disabled={busy} onClick={()=>void save()}>{busy?'Сохранение…':'Сохранить'}</button></div>
  </Modal>}
 </section>;
}
