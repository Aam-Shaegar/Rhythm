import type { PriorityResponse, Task } from '../api/types';
import { CATEGORY_LABELS, PRIORITY_LABELS } from '../tasks';

export default function PriorityOverview({data,tasks,onOpen}: {data: PriorityResponse; tasks: Task[]; onOpen:(task:Task)=>void}) {
 const byID=new Map(tasks.filter(t=>!t.is_completed).map(t=>[t.id,t]));
 const predictions=new Map(data.items.map(p=>[p.task_id,p]));
 const recommended=data.recommendations.map(id=>byID.get(id)).filter((t):t is Task=>!!t);
 const history=data.categories.filter(c=>c.known_outcomes>0);
 return <section className="priority-overview" aria-label="Рекомендации по задачам">
  <div className="priority-heading"><div><h2>С чего начать</h2><p>{data.personal_samples===0?'Базовая оценка · личная история ещё накапливается':`Учтено личных исходов: ${data.personal_samples}`}</p></div><span className="ml-mark">ML</span></div>
  <p className="priority-explanation">Учитываем срок, важность и риск просрочки. Начальная модель обучена на синтетических данных; её оценки уточняются по вашей истории.</p>
  {recommended.length>0 ? <ol className="priority-recommendations">{recommended.map(task=>{
   const p=predictions.get(task.id)!;
   return <li key={task.id}><button onClick={()=>onOpen(task)} className="priority-suggestion"><span className="suggestion-title">{task.title}</span><span className={`priority-badge ${p.level}`}>{PRIORITY_LABELS[p.level]} приоритет</span><span className="suggestion-reason">{p.reasons.join(' · ')}</span></button></li>;
  })}</ol>:<p className="settings-hint">Активных задач для рекомендации нет.</p>}
  <details className="priority-history"><summary>Какие задачи чаще просрочиваются</summary>
   {history.length===0?<p className="settings-hint">Истории пока недостаточно. Здесь появятся категории с известными исходами задач.</p>:<ul>{history.map(c=><li key={c.category}><div><strong>{CATEGORY_LABELS[c.category]??'Прочее'}</strong><span>Просрочено {c.late_outcomes} из {c.known_outcomes}{c.known_outcomes<3?' · мало истории':''}</span></div><p>Выполнено с опозданием: {c.completed_late}; ещё не выполнено: {c.open_late}</p></li>)}</ul>}
   <p className="settings-hint">Будущие незавершённые задачи не считаются ни успехом, ни просрочкой.</p>
  </details>
 </section>;
}
