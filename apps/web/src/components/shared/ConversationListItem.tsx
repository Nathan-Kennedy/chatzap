import { cn } from '@/lib/utils'
import type { Conversation } from '@/types/conversation'
import { formatRelativeShort } from '@/utils/format'
import { avatarColorClass, initialsFromName } from '@/utils/initials'
import { Badge } from '@/components/ui/badge'

type Props = {
  conversation: Conversation
  active: boolean
  onSelect: () => void
}

export function ConversationListItem({
  conversation: c,
  active,
  onSelect,
}: Props) {
  const av = avatarColorClass(c.contact_name)
  const ini = initialsFromName(c.contact_name)

  return (
    <button
      type="button"
      onClick={onSelect}
      className={cn(
        'w-full text-left p-3 border-b border-border flex gap-3 relative transition-colors',
        active
          ? 'bg-card-hover border-l-2 border-l-primary'
          : 'border-l-2 border-l-transparent hover:bg-card'
      )}
    >
      <div
        className={cn(
          'w-10 h-10 rounded-full flex items-center justify-center shrink-0 text-white font-medium text-sm',
          av
        )}
      >
        {ini}
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex justify-between items-baseline mb-0.5">
          <h4 className="font-medium text-sm truncate text-text-primary">
            {c.contact_name}
          </h4>
          <span
            className={cn(
              'text-[11px] shrink-0 ml-1',
              active ? 'text-primary font-medium' : 'text-text-muted'
            )}
          >
            {formatRelativeShort(c.updated_at)}
          </span>
        </div>
        <p className="text-xs text-text-muted truncate">{c.last_message_preview}</p>
      </div>
      {c.unread_count > 0 && (
        <Badge className="absolute top-3 right-3 h-5 min-w-5 px-1 flex items-center justify-center rounded-full bg-primary text-[10px]">
          {c.unread_count}
        </Badge>
      )}
    </button>
  )
}
