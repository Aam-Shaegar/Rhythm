import type { Notice } from '../notifications';
import type { Tab } from '../nav';

const KIND_LABEL: Record<Notice['kind'], string> = {
  overdue: 'Просрочено',
  'soon-event': 'Скоро событие',
  'soon-task': 'Скоро срок',
};

export default function NotificationsPanel({
  open,
  notices,
  onClose,
  onGo,
}: {
  open: boolean;
  notices: Notice[];
  onClose: () => void;
  onGo: (t: Tab) => void;
}) {
  if (!open) return null;
  return (
    <div
      className="notif-backdrop"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="notif-drawer" role="dialog" aria-modal="true" aria-label="Уведомления">
        <div className="modal-head">
          <div className="modal-title">Уведомления{notices.length > 0 ? ` · ${notices.length}` : ''}</div>
          <button className="icon-btn" onClick={onClose} aria-label="Закрыть уведомления">
            ✕
          </button>
        </div>
        {notices.length === 0 ? (
          <div className="state-block">
            <div className="state-title">Всё спокойно</div>
            <div className="state-text">Предстоящих событий и просроченных задач нет.</div>
          </div>
        ) : (
          <div className="notif-list">
            {notices.map((n) => (
              <button
                key={n.id}
                className={`notif-item ${n.kind}`}
                onClick={() => {
                  onGo(n.tab);
                  onClose();
                }}
              >
                <span className="notif-kind">{KIND_LABEL[n.kind]}</span>
                <span className="notif-title">{n.title}</span>
                <span className="notif-detail">{n.detail}</span>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
