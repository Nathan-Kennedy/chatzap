import { useState } from 'react'
import { cn } from '@/lib/utils'
import { sanitizeMessageBody } from '@/lib/sanitize'
import type { Message } from '@/types/conversation'
import { formatMessageTime } from '@/utils/format'
import { avatarColorClass, initialsFromName } from '@/utils/initials'
import { FileText, ImageIcon, Mic, Sparkles, Video } from 'lucide-react'
import { useMessageAttachmentBlobUrl } from '@/hooks/useMessageAttachmentBlobUrl'
import { VoiceNotePlayer } from '@/components/shared/VoiceNotePlayer'
import { Skeleton } from '@/components/ui/skeleton'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'

type Props = {
  message: Message
  contactName: string
}

const MEDIA_PLACEHOLDERS = new Set([
  '[imagem]',
  '[vídeo]',
  '[video]',
  '[áudio]',
  '[documento]',
  '[sticker]',
  '[mídia]',
])

function isMediaPlaceholder(body: string) {
  return MEDIA_PLACEHOLDERS.has(body.trim())
}

export function MessageBubble({ message: m, contactName }: Props) {
  const [lightboxOpen, setLightboxOpen] = useState(false)

  if (m.direction === 'system') {
    return (
      <div className="flex justify-center my-2">
        <span className="text-xs text-text-muted italic">{m.body}</span>
      </div>
    )
  }

  if (m.is_private_note) {
    return (
      <div className="max-w-[80%] rounded-lg border border-warning/40 bg-warning/10 px-3 py-2 text-sm text-text-primary">
        <span className="text-[10px] text-warning font-medium">Nota privada</span>
        <p className="mt-1">{sanitizeMessageBody(m.body)}</p>
      </div>
    )
  }

  const inbound = m.direction === 'inbound'
  const av = avatarColorClass(contactName)
  const ini = initialsFromName(contactName)
  const mt = (m.message_type || 'text').toLowerCase()
  const outbound = !inbound

  /** Só pedir bytes quando a API indica caminho servível (evita rajadas de 404 em histórico sem wa_json/key). */
  const shouldFetchMedia = m.has_attachment === true

  const attachmentBlobHint =
    mt === 'audio'
      ? (m.mime_type?.trim() || 'audio/ogg')
      : mt === 'video'
        ? (m.mime_type?.trim() || 'video/mp4')
        : m.mime_type?.trim()

  const { url: mediaUrl, loading: mediaLoading, failed: mediaFailed } = useMessageAttachmentBlobUrl(
    m.conversation_id,
    m.id,
    shouldFetchMedia,
    attachmentBlobHint,
  )

  const caption =
    m.body && !isMediaPlaceholder(m.body) ? (
      <p className="whitespace-pre-wrap break-words mt-2 text-sm">{sanitizeMessageBody(m.body)}</p>
    ) : null

  const bodyBlock = (() => {
    if (mt === 'image' || mt === 'sticker') {
      return (
        <div className="space-y-1">
          {shouldFetchMedia ? (
            mediaLoading ? (
              <Skeleton className={cn('w-48 rounded-lg', mt === 'sticker' ? 'h-32' : 'h-40')} />
            ) : mediaUrl && !mediaFailed ? (
              <button
                type="button"
                onClick={() => setLightboxOpen(true)}
                className="block rounded-lg overflow-hidden focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
              >
                <img
                  src={mediaUrl}
                  alt=""
                  className={cn(
                    'max-w-[min(100%,280px)] object-cover cursor-zoom-in',
                    mt === 'sticker' ? 'max-h-36 w-36' : 'max-h-52',
                  )}
                />
              </button>
            ) : (
              <div className="flex items-center gap-2 text-xs opacity-90">
                <ImageIcon className="size-4 shrink-0" />
                <span>{m.file_name ? sanitizeMessageBody(m.file_name) : 'Imagem'}</span>
              </div>
            )
          ) : (
            <div className="flex items-center gap-2 text-xs opacity-90">
              <ImageIcon className="size-4 shrink-0" />
              <span>{m.file_name ? sanitizeMessageBody(m.file_name) : mt === 'sticker' ? 'Sticker' : 'Imagem'}</span>
            </div>
          )}
          {caption}
        </div>
      )
    }
    if (mt === 'video') {
      return (
        <div className="space-y-1">
          {shouldFetchMedia ? (
            mediaLoading ? (
              <Skeleton className="w-56 h-36 rounded-lg" />
            ) : mediaUrl && !mediaFailed ? (
              <video
                src={mediaUrl}
                controls
                className="max-w-[min(100%,320px)] max-h-52 rounded-lg bg-black/40"
                preload="metadata"
              />
            ) : (
              <div className="flex items-center gap-2 text-xs opacity-90">
                <Video className="size-4 shrink-0" />
                <span>{m.file_name ? sanitizeMessageBody(m.file_name) : 'Vídeo'}</span>
              </div>
            )
          ) : (
            <div className="flex items-center gap-2 text-xs opacity-90">
              <Video className="size-4 shrink-0" />
              <span>{m.file_name ? sanitizeMessageBody(m.file_name) : 'Vídeo'}</span>
            </div>
          )}
          {caption}
        </div>
      )
    }
    if (mt === 'audio') {
      return (
        <div className="space-y-1">
          {shouldFetchMedia ? (
            mediaLoading ? (
              <Skeleton className="h-14 w-full max-w-[260px] rounded-lg" />
            ) : mediaUrl && !mediaFailed ? (
              <VoiceNotePlayer src={mediaUrl} outbound={outbound} />
            ) : (
              <div className="flex items-center gap-2 text-xs opacity-90">
                <Mic className="size-4 shrink-0" />
                <span>{m.file_name ? sanitizeMessageBody(m.file_name) : 'Mensagem de voz'}</span>
              </div>
            )
          ) : (
            <div className="flex items-center gap-2 text-xs opacity-90">
              <Mic className="size-4 shrink-0" />
              <span>{m.file_name ? sanitizeMessageBody(m.file_name) : 'Mensagem de voz'}</span>
            </div>
          )}
          {caption}
        </div>
      )
    }
    if (mt === 'document') {
      return (
        <div className="space-y-2">
          <div className="flex items-center gap-2 text-xs opacity-90">
            <FileText className="size-4 shrink-0" />
            <span>{m.file_name ? sanitizeMessageBody(m.file_name) : sanitizeMessageBody(m.body)}</span>
          </div>
          {shouldFetchMedia ? (
            mediaLoading ? (
              <Skeleton className="h-9 w-40 rounded-md" />
            ) : mediaUrl && !mediaFailed ? (
              <a
                href={mediaUrl}
                download={m.file_name || 'documento'}
                className={cn(
                  'inline-flex text-xs font-medium underline-offset-2 hover:underline',
                  outbound ? 'text-white/90' : 'text-primary',
                )}
              >
                Descarregar ficheiro
              </a>
            ) : (
              <span className="text-[11px] opacity-80">Não foi possível carregar o ficheiro.</span>
            )
          ) : null}
          {m.body && m.file_name && m.body !== m.file_name && !isMediaPlaceholder(m.body) ? (
            <p className="whitespace-pre-wrap break-words">{sanitizeMessageBody(m.body)}</p>
          ) : null}
        </div>
      )
    }
    return <p className="whitespace-pre-wrap break-words">{sanitizeMessageBody(m.body)}</p>
  })()

  return (
    <>
      <div
        className={cn(
          'flex gap-3 max-w-[80%]',
          inbound ? '' : 'self-end flex-row-reverse',
        )}
      >
        {inbound && (
          <div
            className={cn(
              'w-8 h-8 rounded-full shrink-0 mt-auto flex items-center justify-center text-white text-xs',
              av,
            )}
          >
            {ini}
          </div>
        )}
        <div
          className={cn(
            'p-3 rounded-2xl text-sm border',
            inbound
              ? 'bg-card text-text-primary rounded-bl-sm border-border'
              : 'bg-primary/90 text-white rounded-br-sm border-primary/30',
          )}
        >
          {bodyBlock}
          <div className="flex items-center justify-end gap-1.5 mt-1">
            {m.is_ai && (
              <span className="text-[10px] opacity-80 flex items-center gap-0.5">
                <Sparkles size={10} /> IA
              </span>
            )}
            <span
              className={cn(
                'text-[10px]',
                inbound ? 'text-text-muted' : 'text-white/70',
              )}
            >
              {formatMessageTime(m.created_at)}
            </span>
          </div>
        </div>
      </div>

      <Dialog open={lightboxOpen} onOpenChange={setLightboxOpen}>
        <DialogContent className="max-w-[95vw] w-auto p-2 bg-black/95 border-border">
          <DialogHeader className="sr-only">
            <DialogTitle>Pré-visualização da imagem</DialogTitle>
          </DialogHeader>
          {mediaUrl ? (
            <img src={mediaUrl} alt="" className="max-h-[85vh] max-w-full object-contain mx-auto" />
          ) : null}
        </DialogContent>
      </Dialog>
    </>
  )
}
