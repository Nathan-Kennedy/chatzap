import { useEffect, useMemo, useRef, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import {
  Search,
  MessageSquare,
  Phone,
  MoreVertical,
  Paperclip,
  Mic,
  Send,
  Smile,
  MessageCirclePlus,
  Download,
  RefreshCw,
  Trash2,
} from 'lucide-react'
import { useConversations } from '@/hooks/useConversations'
import { useConversationMessages } from '@/hooks/useConversationMessages'
import { useInstances } from '@/hooks/useInstances'
import { ConversationListItem } from '@/components/shared/ConversationListItem'
import { MessageBubble } from '@/components/shared/MessageBubble'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Skeleton } from '@/components/ui/skeleton'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Badge } from '@/components/ui/badge'
import { formatRelativeShort } from '@/utils/format'
import { api, postMultipart, unwrapEnvelope } from '@/lib/api'
import { ApiEnvelopeError } from '@/types/api'
import { toast } from 'sonner'
import { avatarColorClass, initialsFromName } from '@/utils/initials'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Label } from '@/components/ui/label'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Progress } from '@/components/ui/progress'
import type { Conversation, MessageType } from '@/types/conversation'
import { CallContactModal } from '@/components/shared/CallContactModal'

const QUICK_EMOJIS = [
  '😀',
  '😂',
  '😍',
  '🥰',
  '😉',
  '😎',
  '🤔',
  '👍',
  '👎',
  '👏',
  '🙏',
  '🔥',
  '✨',
  '❤️',
  '💯',
  '✅',
  '❌',
  '⭐',
  '🎉',
  '📎',
  '📷',
  '🎤',
  '📞',
]

function inferMediaKind(mime: string): MessageType {
  const m = mime.toLowerCase()
  if (m.startsWith('image/')) return 'image'
  if (m.startsWith('video/')) return 'video'
  if (m.startsWith('audio/')) return 'audio'
  return 'document'
}

export default function Inbox() {
  const qc = useQueryClient()
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [draft, setDraft] = useState('')
  const [newChatOpen, setNewChatOpen] = useState(false)
  const [newPhone, setNewPhone] = useState('')
  const [newName, setNewName] = useState('')
  /** Instância usada na nova conversa e em “Importar do WhatsApp”. */
  const [pickerInstanceId, setPickerInstanceId] = useState('')
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [callDialogOpen, setCallDialogOpen] = useState(false)
  const [emojiOpen, setEmojiOpen] = useState(false)
  const [mediaUploadPct, setMediaUploadPct] = useState<number | null>(null)
  const [recording, setRecording] = useState(false)
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const mediaStreamRef = useRef<MediaStream | null>(null)
  const audioChunksRef = useRef<Blob[]>([])

  useEffect(() => {
    const t = setTimeout(() => setDebouncedSearch(search), 300)
    return () => clearTimeout(t)
  }, [search])

  const { data: conversations = [], isLoading } =
    useConversations(debouncedSearch)
  const { data: instances = [], isLoading: loadingInstances } = useInstances()

  useEffect(() => {
    if (instances.length > 0 && !pickerInstanceId) {
      setPickerInstanceId(instances[0].id)
    }
  }, [instances, pickerInstanceId])

  const effectiveInstanceId = pickerInstanceId || instances[0]?.id || ''

  const effectiveId = selectedId ?? conversations[0]?.id ?? null

  const { data: messages = [], isLoading: loadingMessages } =
    useConversationMessages(effectiveId)

  const selected = useMemo(
    () => conversations.find((c) => c.id === effectiveId) ?? null,
    [conversations, effectiveId]
  )

  const mineCount = conversations.filter((c) => c.assigned_agent_initials).length

  const createConvMut = useMutation({
    mutationFn: async (vars: { instanceId: string; phone: string; contactName: string }) => {
      const res = await api.post<unknown>('/conversations', {
        whatsapp_instance_id: vars.instanceId,
        phone: vars.phone,
        contact_name: vars.contactName || undefined,
      })
      return unwrapEnvelope<Conversation>(res).data
    },
    onSuccess: (data) => {
      toast.success('Conversa criada')
      setNewChatOpen(false)
      setSelectedId(data.id)
      void qc.invalidateQueries({ queryKey: ['conversations'] })
    },
    onError: (err: unknown) => {
      if (err instanceof ApiEnvelopeError) {
        if (err.code === 'conversation_exists') {
          const det = err.details as { conversation?: { id?: string } } | undefined
          const existingId =
            det?.conversation && typeof det.conversation.id === 'string'
              ? det.conversation.id
              : undefined
          toast.message('Já existe conversa com este contacto.')
          void qc.invalidateQueries({ queryKey: ['conversations'] })
          setNewChatOpen(false)
          if (existingId) setSelectedId(existingId)
          return
        }
        toast.error(err.message)
      } else toast.error('Não foi possível criar a conversa')
    },
  })

  const syncContactsMut = useMutation({
    mutationFn: async (instanceId: string) => {
      const res = await api.post<unknown>(`/instances/${instanceId}/sync-contacts`)
      return unwrapEnvelope<{
        total_fetched: number
        created: number
        already_existing: number
        skipped: number
      }>(res).data
    },
    onSuccess: (d) => {
      if (d.created > 0) {
        toast.success(
          `${d.created} conversa(s) nova(s). ${d.already_existing} já estavam na caixa. (${d.total_fetched} entradas no telefone / Evolution)`
        )
      } else {
        toast.message(
          `Nenhuma conversa nova. ${d.already_existing} já existiam; ${d.skipped} ignoradas; ${d.total_fetched} lidas da Evolution.`
        )
      }
      void qc.invalidateQueries({ queryKey: ['conversations'] })
    },
    onError: (err: unknown) => {
      if (err instanceof ApiEnvelopeError) toast.error(err.message)
      else toast.error('Não foi possível importar os chats do WhatsApp')
    },
  })

  const recoverMessagesMut = useMutation({
    mutationFn: async (vars: { instanceId: string; contactJid: string }) => {
      // 1º Evolution (histórico antigo ordenado no servidor), 2º webhooks (eventos já guardados) — menos confusão na timeline.
      let sync = { inserted: 0, parsed: 0 }
      try {
        const syncRes = await api.post<unknown>(`/instances/${vars.instanceId}/sync-chats`, {
          contact_jid: vars.contactJid,
        })
        sync = unwrapEnvelope<{ inserted: number; parsed: number }>(syncRes).data
      } catch (e: unknown) {
        if (e instanceof ApiEnvelopeError && e.code === 'sync_not_supported') {
          // Evolution sem findMessages; só o passo de webhook importa.
        } else {
          throw e
        }
      }
      const rec = await api.post<unknown>(`/instances/${vars.instanceId}/reconcile-inbox`, {
        limit: 500,
      })
      const reconcile = unwrapEnvelope<{ new_messages: number; scanned: number }>(rec).data
      return { reconcile, sync }
    },
    onSuccess: (d) => {
      const parts: string[] = []
      if (d.sync.inserted > 0) {
        parts.push(`${d.sync.inserted} da Evolution (${d.sync.parsed} lidas)`)
      }
      if (d.reconcile.new_messages > 0) {
        parts.push(`${d.reconcile.new_messages} de webhooks (${d.reconcile.scanned} eventos revistos)`)
      }
      if (parts.length === 0) {
        toast.message(
          'Nada novo: webhooks já processados ou Evolution sem findMessages / sem histórico para este JID.'
        )
      } else {
        toast.success(parts.join(' · '))
      }
      void qc.invalidateQueries({ queryKey: ['conversations'] })
      if (effectiveId) {
        void qc.invalidateQueries({ queryKey: ['conversation', effectiveId, 'messages'] })
        void qc.refetchQueries({ queryKey: ['conversation', effectiveId, 'messages'] })
      }
    },
    onError: (err: unknown) => {
      if (err instanceof ApiEnvelopeError) toast.error(err.message)
      else toast.error('Não foi possível recuperar mensagens')
    },
  })

  const deleteConvMut = useMutation({
    mutationFn: async (conversationId: string) => {
      const res = await api.delete<unknown>(`/conversations/${conversationId}`)
      unwrapEnvelope<{ ok?: boolean }>(res)
    },
    onSuccess: (_, deletedId) => {
      toast.success('Conversa e histórico removidos')
      setDeleteDialogOpen(false)
      if (selectedId === deletedId) setSelectedId(null)
      void qc.invalidateQueries({ queryKey: ['conversations'] })
      void qc.removeQueries({ queryKey: ['conversation', deletedId, 'messages'] })
    },
    onError: (err: unknown) => {
      if (err instanceof ApiEnvelopeError) toast.error(err.message)
      else toast.error('Não foi possível excluir a conversa')
    },
  })

  const sendMut = useMutation({
    mutationFn: async (text: string) => {
      if (!effectiveId) throw new Error('sem conversa')
      await api.post(`/conversations/${effectiveId}/messages`, { body: text })
    },
    onSuccess: () => {
      setDraft('')
      void qc.invalidateQueries({ queryKey: ['conversation', effectiveId, 'messages'] })
      void qc.invalidateQueries({ queryKey: ['conversations'] })
    },
    onError: (err: unknown) => {
      if (err instanceof ApiEnvelopeError) toast.error(err.message)
      else toast.error('Não foi possível enviar (Evolution configurado?)')
    },
  })

  const sendMediaMut = useMutation({
    mutationFn: async (vars: { file: File; kind: MessageType }) => {
      if (!effectiveId) throw new Error('sem conversa')
      const fd = new FormData()
      fd.append('file', vars.file)
      fd.append('type', vars.kind)
      const cap = draft.trim()
      if (cap) fd.append('caption', cap)
      await postMultipart<{ ok: boolean }>(
        `/conversations/${effectiveId}/messages/media`,
        fd,
        (p) => setMediaUploadPct(p),
      )
    },
    onSuccess: () => {
      setDraft('')
      setMediaUploadPct(null)
      void qc.invalidateQueries({ queryKey: ['conversation', effectiveId, 'messages'] })
      void qc.invalidateQueries({ queryKey: ['conversations'] })
      toast.success('Mídia enviada')
    },
    onError: (err: unknown) => {
      setMediaUploadPct(null)
      if (err instanceof ApiEnvelopeError) toast.error(err.message)
      else toast.error('Não foi possível enviar o ficheiro')
    },
  })

  function insertEmoji(ch: string) {
    const el = textareaRef.current
    if (!el) {
      setDraft((d) => d + ch)
      return
    }
    const start = el.selectionStart ?? draft.length
    const end = el.selectionEnd ?? draft.length
    const next = draft.slice(0, start) + ch + draft.slice(end)
    setDraft(next)
    requestAnimationFrame(() => {
      el.focus()
      const pos = start + ch.length
      el.setSelectionRange(pos, pos)
    })
    setEmojiOpen(false)
  }

  async function stopRecordingAndSend() {
    const rec = mediaRecorderRef.current
    if (!rec || rec.state === 'inactive') {
      setRecording(false)
      mediaRecorderRef.current = null
      mediaStreamRef.current?.getTracks().forEach((t) => t.stop())
      mediaStreamRef.current = null
      return
    }
    await new Promise<void>((resolve) => {
      rec.addEventListener('stop', () => resolve(), { once: true })
      rec.stop()
    })
    mediaStreamRef.current?.getTracks().forEach((t) => t.stop())
    mediaStreamRef.current = null
    mediaRecorderRef.current = null
    setRecording(false)
    const blob = new Blob(audioChunksRef.current, { type: 'audio/webm' })
    audioChunksRef.current = []
    if (blob.size < 64) {
      toast.message('Gravação demasiado curta')
      return
    }
    const file = new File([blob], `voice-${Date.now()}.webm`, { type: blob.type || 'audio/webm' })
    sendMediaMut.mutate({ file, kind: 'audio' })
  }

  function startRecording() {
    if (recording) {
      void stopRecordingAndSend()
      return
    }
    if (!effectiveId || sendMediaMut.isPending) return
    void navigator.mediaDevices
      .getUserMedia({ audio: true })
      .then((stream) => {
        mediaStreamRef.current = stream
        audioChunksRef.current = []
        const mime = MediaRecorder.isTypeSupported('audio/webm') ? 'audio/webm' : ''
        const mr = mime ? new MediaRecorder(stream, { mimeType: mime }) : new MediaRecorder(stream)
        mediaRecorderRef.current = mr
        mr.ondataavailable = (e) => {
          if (e.data.size > 0) audioChunksRef.current.push(e.data)
        }
        mr.start(200)
        setRecording(true)
        toast.message('A gravar… clica de novo no microfone para enviar')
      })
      .catch(() => {
        toast.error('Não foi possível aceder ao microfone')
      })
  }

  return (
    <div className="flex h-full w-full min-h-0">
      <div className="w-[320px] bg-background border-r border-border flex flex-col shrink-0 text-text-primary min-h-0">
        <div className="p-4 border-b border-border shrink-0">
          <div className="flex items-center justify-between mb-3 gap-2">
            <h2 className="font-semibold text-lg">Caixa de Entrada</h2>
            <Dialog
              open={newChatOpen}
              onOpenChange={(o) => {
                setNewChatOpen(o)
                if (o) {
                  void qc.invalidateQueries({ queryKey: ['instances'] })
                  if (instances[0]?.id && !pickerInstanceId) {
                    setPickerInstanceId(instances[0].id)
                  }
                }
              }}
            >
              <DialogTrigger asChild>
                <Button size="sm" className="gap-1 shrink-0 bg-primary" disabled={!instances.length}>
                  <MessageCirclePlus className="size-4" />
                  Nova
                </Button>
              </DialogTrigger>
              <DialogContent className="bg-card border-border sm:max-w-md">
                <DialogHeader>
                  <DialogTitle>Nova conversa</DialogTitle>
                </DialogHeader>
                <div className="space-y-4 py-2">
                  {!instances.length && !loadingInstances ? (
                    <p className="text-sm text-text-muted">
                      Nenhuma instância WhatsApp.{' '}
                      <Link to="/instances" className="text-primary underline">
                        Configurar instâncias
                      </Link>
                    </p>
                  ) : (
                    <>
                      <div className="space-y-1.5">
                        <Label htmlFor="inbox-inst">Instância</Label>
                        <select
                          id="inbox-inst"
                          className="w-full h-10 rounded-md border border-border bg-background px-3 text-sm"
                          value={effectiveInstanceId}
                          onChange={(e) => setPickerInstanceId(e.target.value)}
                        >
                          {instances.map((i) => (
                            <option key={i.id} value={i.id}>
                              {i.name} ({i.evolution_instance_name ?? i.id.slice(0, 8)})
                            </option>
                          ))}
                        </select>
                      </div>
                      <div className="space-y-1.5">
                        <Label htmlFor="inbox-phone">Telefone (com DDD)</Label>
                        <Input
                          id="inbox-phone"
                          placeholder="69993378283 ou 5569993378283"
                          value={newPhone}
                          onChange={(e) => setNewPhone(e.target.value)}
                          className="bg-background"
                        />
                        <p className="text-[11px] text-text-muted">
                          Será normalizado para JID (ex. Brasil +55).
                        </p>
                      </div>
                      <div className="space-y-1.5">
                        <Label htmlFor="inbox-name">Nome do contacto (opcional)</Label>
                        <Input
                          id="inbox-name"
                          value={newName}
                          onChange={(e) => setNewName(e.target.value)}
                          className="bg-background"
                        />
                      </div>
                      <Button
                        className="w-full"
                        disabled={
                          createConvMut.isPending ||
                          !effectiveInstanceId ||
                          !newPhone.trim()
                        }
                        onClick={() => {
                          if (!effectiveInstanceId || !newPhone.trim()) return
                          createConvMut.mutate({
                            instanceId: effectiveInstanceId,
                            phone: newPhone.trim(),
                            contactName: newName.trim(),
                          })
                        }}
                      >
                        {createConvMut.isPending ? 'A criar…' : 'Criar conversa'}
                      </Button>
                    </>
                  )}
                </div>
              </DialogContent>
            </Dialog>
            <Button
              type="button"
              size="sm"
              variant="outline"
              className="gap-1 shrink-0 border-border"
              disabled={
                !effectiveInstanceId || syncContactsMut.isPending || !instances.length
              }
              title="Cria conversas na caixa a partir dos contactos/chats no aparelho (Evolution GET /user/contacts). Não traz texto de mensagens antigas."
              onClick={() => syncContactsMut.mutate(effectiveInstanceId)}
            >
              <Download
                className={`size-4 ${syncContactsMut.isPending ? 'animate-pulse' : ''}`}
              />
              Importar chats
            </Button>
          </div>
          {instances.length > 0 ? (
            <div className="mb-3 space-y-1">
              <Label htmlFor="inbox-picker-inst" className="text-[11px] text-text-muted">
                Instância (nova conversa e importar)
              </Label>
              <select
                id="inbox-picker-inst"
                className="w-full h-9 rounded-md border border-border bg-card px-2 text-sm"
                value={effectiveInstanceId}
                onChange={(e) => setPickerInstanceId(e.target.value)}
              >
                {instances.map((i) => (
                  <option key={i.id} value={i.id}>
                    {i.name} ({i.evolution_instance_name ?? i.id.slice(0, 8)})
                  </option>
                ))}
              </select>
            </div>
          ) : null}
          <div className="flex gap-4 border-b border-border mb-3 text-sm font-medium">
            <button
              type="button"
              className="text-primary border-b-2 border-primary pb-2 px-1"
            >
              Minhas ({mineCount})
            </button>
            <button
              type="button"
              className="text-text-muted hover:text-text-secondary pb-2 px-1"
            >
              Não atribuídas
            </button>
          </div>
          <div className="relative">
            <Search
              className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted"
              size={16}
            />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Buscar conversas..."
              className="pl-9 bg-card border-border"
            />
          </div>
        </div>

        <ScrollArea className="flex-1">
          {isLoading ? (
            <div className="p-3 space-y-3">
              <Skeleton className="h-16 w-full bg-card" />
              <Skeleton className="h-16 w-full bg-card" />
            </div>
          ) : conversations.length === 0 ? (
            <div className="p-6 text-center text-sm text-text-muted">
              <p>Sem conversas ainda.</p>
              <p className="mt-2 text-xs">
                Mensagens recebidas pelo webhook aparecem aqui, ou abre uma conversa nova.
              </p>
            </div>
          ) : (
            conversations.map((c) => (
              <ConversationListItem
                key={c.id}
                conversation={c}
                active={c.id === effectiveId}
                onSelect={() => setSelectedId(c.id)}
              />
            ))
          )}
        </ScrollArea>
      </div>

      <div className="flex-1 flex flex-col min-w-0 bg-background text-text-primary min-h-0">
        {selected ? (
          <>
            <div className="h-[68px] border-b border-border flex items-center justify-between px-6 shrink-0 bg-card">
              <div className="flex items-center gap-3 min-w-0">
                <div
                  className={`w-10 h-10 rounded-full flex items-center justify-center text-white font-medium shrink-0 ${avatarColorClass(selected.contact_name)}`}
                >
                  {initialsFromName(selected.contact_name)}
                </div>
                <div className="min-w-0">
                  <h3 className="font-medium text-sm leading-tight truncate">
                    {selected.contact_name}
                  </h3>
                  <div className="flex items-center gap-1.5 mt-0.5">
                    <span className="w-2 h-2 rounded-full bg-success shrink-0" />
                    <span className="text-xs text-text-muted truncate">
                      {selected.contact_phone}
                    </span>
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-3 shrink-0">
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  className="gap-1 border-border"
                  disabled={
                    recoverMessagesMut.isPending ||
                    !(selected.whatsapp_instance_id || effectiveInstanceId) ||
                    !selected.contact_id
                  }
                  title="Reprocessa os últimos eventos de webhook desta instância (ordenados pela data da mensagem) e, se a Evolution tiver findMessages, importa histórico extra — sem alterar a ordem do chat na app."
                  onClick={() => {
                    const iid = selected.whatsapp_instance_id || effectiveInstanceId
                    if (!iid || !selected.contact_id) return
                    recoverMessagesMut.mutate({
                      instanceId: iid,
                      contactJid: selected.contact_id,
                    })
                  }}
                >
                  <RefreshCw
                    className={`size-4 ${recoverMessagesMut.isPending ? 'animate-spin' : ''}`}
                  />
                  Recuperar
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  className="text-success border-success/30 bg-success/10 hover:bg-success/20"
                >
                  Resolver
                </Button>
                <div className="w-px h-6 bg-border mx-1" />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="text-text-muted"
                  title="Ligar / WhatsApp"
                  onClick={() => setCallDialogOpen(true)}
                >
                  <Phone size={18} />
                </Button>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="icon" className="text-text-muted">
                      <MoreVertical size={18} />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" className="bg-card border-border">
                    <DropdownMenuItem
                      className="text-destructive focus:text-destructive cursor-pointer"
                      onSelect={(e) => {
                        e.preventDefault()
                        setDeleteDialogOpen(true)
                      }}
                    >
                      <Trash2 className="mr-2 size-4" />
                      Excluir conversa…
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </div>

            <ScrollArea className="flex-1 p-6">
              <div className="flex flex-col gap-4 max-w-3xl mx-auto">
                <div className="flex justify-center my-2">
                  <span className="bg-card px-3 py-1 rounded-full text-xs text-text-muted border border-border">
                    Hoje
                  </span>
                </div>
                {loadingMessages ? (
                  <Skeleton className="h-20 w-full max-w-md bg-card" />
                ) : (
                  messages.map((m) => (
                    <MessageBubble
                      key={m.id}
                      message={m}
                      contactName={selected.contact_name}
                    />
                  ))
                )}
              </div>
            </ScrollArea>

            <div className="p-4 bg-card border-t border-border shrink-0">
              <input
                ref={fileInputRef}
                type="file"
                className="hidden"
                accept="image/*,video/*,audio/*,.pdf,.doc,.docx,.xls,.xlsx,.zip"
                onChange={(e) => {
                  const f = e.target.files?.[0]
                  e.target.value = ''
                  if (!f || !effectiveId) return
                  sendMediaMut.mutate({ file: f, kind: inferMediaKind(f.type || '') })
                }}
              />
              <div className="flex gap-4 mb-2">
                <button
                  type="button"
                  className="text-xs font-medium text-primary border-b-2 border-primary pb-1"
                >
                  Responder
                </button>
                <button
                  type="button"
                  className="text-xs font-medium text-text-muted pb-1"
                >
                  Nota Privada
                </button>
              </div>
              {mediaUploadPct !== null ? (
                <div className="mb-2 space-y-1">
                  <Progress value={mediaUploadPct} className="h-2" />
                  <p className="text-[11px] text-text-muted">A enviar mídia… {mediaUploadPct}%</p>
                </div>
              ) : null}
              <div className="rounded-xl border border-border bg-background focus-within:ring-1 focus-within:ring-primary overflow-hidden">
                <Textarea
                  ref={textareaRef}
                  rows={3}
                  placeholder="Digite uma mensagem... (Use / para respostas rápidas)"
                  className="border-0 bg-transparent focus-visible:ring-0 resize-none"
                  value={draft}
                  onChange={(e) => setDraft(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && !e.shiftKey) {
                      e.preventDefault()
                      const t = draft.trim()
                      if (t) sendMut.mutate(t)
                    }
                  }}
                />
                <div className="flex items-center justify-between p-2 border-t border-border/50 bg-card/50">
                  <div className="flex gap-1 text-text-muted">
                    <Popover open={emojiOpen} onOpenChange={setEmojiOpen}>
                      <PopoverTrigger asChild>
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8"
                          title="Emoji"
                        >
                          <Smile size={18} />
                        </Button>
                      </PopoverTrigger>
                      <PopoverContent
                        className="w-auto p-2 bg-card border-border"
                        align="start"
                        side="top"
                      >
                        <div className="grid grid-cols-7 gap-1 max-w-[220px]">
                          {QUICK_EMOJIS.map((em) => (
                            <button
                              key={em}
                              type="button"
                              className="text-lg p-1 rounded hover:bg-muted"
                              onClick={() => insertEmoji(em)}
                            >
                              {em}
                            </button>
                          ))}
                        </div>
                      </PopoverContent>
                    </Popover>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8"
                      title="Anexar ficheiro"
                      disabled={sendMediaMut.isPending || !effectiveId}
                      onClick={() => fileInputRef.current?.click()}
                    >
                      <Paperclip size={18} />
                    </Button>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className={`h-8 w-8 ${recording ? 'text-destructive' : ''}`}
                      title={recording ? 'Parar e enviar áudio' : 'Gravar nota de voz'}
                      disabled={sendMediaMut.isPending || !effectiveId}
                      onClick={() => startRecording()}
                    >
                      <Mic size={18} />
                    </Button>
                  </div>
                  <Button
                    type="button"
                    className="bg-primary hover:bg-primary-hover gap-2"
                    disabled={
                      sendMut.isPending || sendMediaMut.isPending || !draft.trim()
                    }
                    onClick={() => {
                      const t = draft.trim()
                      if (t) sendMut.mutate(t)
                    }}
                  >
                    <span>Enviar</span>
                    <Send size={14} />
                  </Button>
                </div>
              </div>
            </div>
          </>
        ) : (
          <div className="flex-1 flex flex-col items-center justify-center gap-4 p-8 text-center text-text-muted">
            <MessageSquare className="size-14 opacity-30" />
            <div>
              <p className="text-text-primary font-medium">Nenhuma conversa selecionada</p>
              <p className="text-sm mt-1 max-w-sm">
                Cria uma conversa com o teu número de teste ou aguarda mensagens entrantes (webhook +
                Postgres).
              </p>
            </div>
            <Button className="gap-2" onClick={() => setNewChatOpen(true)} disabled={!instances.length}>
              <MessageCirclePlus className="size-4" />
              Nova conversa
            </Button>
            {!instances.length && !loadingInstances ? (
              <Link to="/instances" className="text-sm text-primary underline">
                Adicionar instância WhatsApp
              </Link>
            ) : null}
          </div>
        )}
      </div>

      <div className="w-[280px] bg-card border-l border-border shrink-0 hidden lg:flex flex-col text-sm text-text-primary overflow-y-auto">
        {selected ? (
          <>
            <div className="p-6 border-b border-border flex flex-col items-center text-center">
              <div
                className={`w-20 h-20 rounded-full flex items-center justify-center text-white text-2xl font-semibold mb-3 ${avatarColorClass(selected.contact_name)}`}
              >
                {initialsFromName(selected.contact_name)}
              </div>
              <h3 className="font-semibold text-base mb-1">
                {selected.contact_name}
              </h3>
              <p className="text-text-muted text-xs mb-4">
                {selected.contact_phone}
              </p>
              <div className="flex gap-2 text-xs flex-wrap justify-center">
                <Badge variant="secondary" className="bg-primary/20 text-primary">
                  Lead
                </Badge>
                <Badge variant="secondary" className="bg-success/20 text-success">
                  VIP
                </Badge>
              </div>
            </div>
            <div className="p-4">
              <h4 className="font-medium text-text-secondary mb-3 text-xs uppercase tracking-wider">
                Informações
              </h4>
              <div className="space-y-3 text-sm">
                <div>
                  <div className="text-text-muted text-xs">Última atividade</div>
                  <div>{formatRelativeShort(selected.updated_at)}</div>
                </div>
                <div>
                  <div className="text-text-muted text-xs">Canal</div>
                  <div className="capitalize">{selected.channel}</div>
                </div>
                {selected.assigned_agent_initials && (
                  <div>
                    <div className="text-text-muted text-xs">Atendente</div>
                    <div className="flex items-center gap-2 mt-1">
                      <div className="w-5 h-5 rounded-full bg-primary flex items-center justify-center text-[10px] text-white">
                        {selected.assigned_agent_initials}
                      </div>
                    </div>
                  </div>
                )}
              </div>
              <div className="p-4 pt-0 mt-2 border-t border-border">
                <Button
                  type="button"
                  variant="outline"
                  className="w-full border-destructive/40 text-destructive hover:bg-destructive/10 hover:text-destructive"
                  onClick={() => setDeleteDialogOpen(true)}
                >
                  <Trash2 className="size-4 mr-2" />
                  Excluir conversa
                </Button>
                <p className="text-[11px] text-text-muted mt-2">
                  Remove esta thread e todas as mensagens na caixa (útil para duplicados).
                </p>
              </div>
            </div>
          </>
        ) : null}
      </div>

      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent className="bg-card border-border sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Excluir conversa?</DialogTitle>
            <DialogDescription className="text-text-muted">
              {selected ? (
                <>
                  Isto apaga <strong className="text-text-primary">{selected.contact_name}</strong> (
                  {selected.contact_phone}) e todo o histórico desta conversa na aplicação. Não apaga chats no
                  telefone nem na Evolution.
                </>
              ) : (
                'Seleciona uma conversa.'
              )}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2 sm:gap-0">
            <Button
              type="button"
              variant="outline"
              className="border-border"
              onClick={() => setDeleteDialogOpen(false)}
              disabled={deleteConvMut.isPending}
            >
              Cancelar
            </Button>
            <Button
              type="button"
              variant="destructive"
              disabled={!effectiveId || deleteConvMut.isPending}
              onClick={() => {
                if (effectiveId) deleteConvMut.mutate(effectiveId)
              }}
            >
              {deleteConvMut.isPending ? 'A excluir…' : 'Excluir'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {selected ? (
        <CallContactModal
          open={callDialogOpen}
          onOpenChange={setCallDialogOpen}
          contactName={selected.contact_name}
          phoneOrJid={selected.contact_phone || selected.contact_id}
        />
      ) : null}
    </div>
  )
}
