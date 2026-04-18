import { useEffect, useMemo, useState } from 'react'
import { useForm, Controller } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Bot, Pencil, Sparkles, Trash2, FlaskConical } from 'lucide-react'
import { api, unwrapEnvelope } from '@/lib/api'
import { ApiEnvelopeError } from '@/types/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { toast } from 'sonner'

// IDs estáveis para v1beta :generateContent — evitar "gemini-2.5-flash-preview" sem data (a API recusa).
// Previews datados: ver https://ai.google.dev/gemini-api/docs/models/gemini
const GEMINI_MODELS = ['gemini-2.5-flash', 'gemini-2.0-flash', 'gemini-1.5-flash'] as const
const OPENAI_MODELS = ['gpt-4o-mini', 'gpt-4o', 'gpt-4-turbo'] as const
const CUSTOM_MODEL = '__custom__'
const VOICE_CUSTOM = '__custom_voice__'

/** Valores de `tts_provider` alinhados ao backend */
type TTSProviderValue = 'none' | 'openai_tts' | 'omnivoice' | 'elevenlabs' | 'kokoro'

const OPENAI_TTS_VOICES: { value: string; label: string }[] = [
  { value: 'coral', label: 'Coral' },
  { value: 'nova', label: 'Nova' },
  { value: 'shimmer', label: 'Shimmer' },
  { value: 'sage', label: 'Sage' },
  { value: 'marin', label: 'Marin' },
  { value: 'cedar', label: 'Cedar' },
  { value: 'alloy', label: 'Alloy' },
  { value: 'ash', label: 'Ash' },
  { value: 'ballad', label: 'Ballad' },
  { value: 'echo', label: 'Echo' },
  { value: 'fable', label: 'Fable' },
  { value: 'onyx', label: 'Onyx' },
  { value: 'verse', label: 'Verse' },
]

/** Vozes femininas curadas (IDs públicos; podem mudar na conta ElevenLabs) */
const ELEVENLABS_FEMALE_VOICES: { value: string; label: string }[] = [
  { value: '21m00Tcm4TlvDq8ikWAM', label: 'Rachel (EN)' },
  { value: 'EXAVITQu4vr4xnSDxMaL', label: 'Bella' },
  { value: 'MF3mGyEYCl7XYWbV9V6O', label: 'Elli' },
  { value: 'ThT5KcBeYPX3keUQqHPh', label: 'Dorothy' },
  { value: 'XB0fDUnXU5powFXDhCwa', label: 'Charlotte' },
]

const KOKORO_VOICE_PRESETS: { value: string; label: string }[] = [
  { value: 'pf_dora', label: 'PT-BR feminina (pf_dora)' },
  { value: 'af_heart', label: 'US feminina (af_heart)' },
  { value: 'bf_emma', label: 'bf_emma' },
]

function defaultVoiceForTTSProvider(p: TTSProviderValue): string {
  switch (p) {
    case 'openai_tts':
      return 'coral'
    case 'omnivoice':
      return 'clone:atendimento_br'
    case 'elevenlabs':
      return '21m00Tcm4TlvDq8ikWAM'
    case 'kokoro':
      return 'pf_dora'
    default:
      return 'nova'
  }
}

function ttsProviderBadgeLabel(tts: string): string {
  switch (tts) {
    case 'openai_tts':
      return 'OpenAI TTS'
    case 'omnivoice':
      return 'OmniVoice'
    case 'elevenlabs':
      return 'ElevenLabs'
    case 'kokoro':
      return 'Kokoro'
    default:
      return tts
  }
}

const INFINITI_EXAMPLE = {
  name: 'Assistente Infiniti',
  role: 'Secretária virtual do escritório Infiniti Engenharia',
  description:
    'Tom cordial e profissional. Atendimento inicial: esclarecer serviços de engenharia civil, agendar visitas e encaminhar pedidos de orçamento para a equipe técnica. Não invente valores nem prazos; quando não souber, ofereça contato humano.',
}

export type AgentRow = {
  id: string
  name: string
  provider: string
  model: string
  has_api_key: boolean
  api_key_last4: string
  role: string
  description: string
  active: boolean
  use_for_whatsapp_auto_reply: boolean
  voice_reply_enabled: boolean
  tts_provider: string
  openai_tts_voice: string
  openai_tts_model: string
  has_openai_tts_api_key: boolean
  openai_tts_api_key_last4: string
  omnivoice_base_url: string
  kokoro_base_url: string
  has_elevenlabs_api_key: boolean
  elevenlabs_api_key_last4: string
  /** Amostra TTS gerada no servidor (GET /agents/:id/voice-preview) */
  voice_preview_available?: boolean
  created_at: string
  updated_at: string
}

const formSchema = z
  .object({
    name: z.string().min(1, 'Nome obrigatório'),
    provider: z.enum(['gemini', 'openai']),
    model_preset: z.string().min(1),
    model_custom: z.string(),
    api_key: z.string(),
    role: z.string(),
    description: z.string(),
    active: z.boolean(),
    use_for_whatsapp_auto_reply: z.boolean(),
    voice_reply_enabled: z.boolean(),
    tts_provider: z.enum(['none', 'openai_tts', 'omnivoice', 'elevenlabs', 'kokoro']),
    openai_tts_voice: z.string().min(1, 'Indica a voz').max(128),
    openai_tts_model: z.string().max(128),
    openai_tts_api_key: z.string(),
    omnivoice_base_url: z.string(),
    kokoro_base_url: z.string(),
    elevenlabs_api_key: z.string(),
  })
  .superRefine((data, ctx) => {
    const model =
      data.model_preset === CUSTOM_MODEL
        ? data.model_custom.trim()
        : data.model_preset.trim()
    if (!model) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: 'Indica o modelo ou escolhe “Personalizado” e preenche.',
        path: ['model_custom'],
      })
    }
    if (data.voice_reply_enabled) {
      if (data.tts_provider === 'none') {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: 'Escolhe um provedor de voz ou desativa a resposta em áudio.',
          path: ['tts_provider'],
        })
      }
      const v = data.openai_tts_voice.trim()
      if (!v) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: 'Indica o identificador da voz.',
          path: ['openai_tts_voice'],
        })
      }
    }
  })

type AgentFormValues = z.infer<typeof formSchema>

function modelsForProvider(p: 'gemini' | 'openai') {
  return p === 'openai' ? OPENAI_MODELS : GEMINI_MODELS
}

function resolveModelPreset(model: string, provider: 'gemini' | 'openai'): { preset: string; custom: string } {
  const list = modelsForProvider(provider) as readonly string[]
  if (list.includes(model)) {
    return { preset: model, custom: '' }
  }
  return { preset: CUSTOM_MODEL, custom: model }
}

function hintFromApiKey(key: string): string | null {
  const t = key.trim()
  if (t.startsWith('sk-') || t.startsWith('sk-proj-')) {
    return 'Esta chave parece ser da OpenAI (ajuda apenas na UI; confirma o fornecedor no menu).'
  }
  if (t.startsWith('AIza')) {
    return 'Esta chave parece ser do Google AI / Gemini.'
  }
  return null
}

function useAgentVoicePreviewUrl(agentId: string | undefined, shouldLoad: boolean) {
  const [audioUrl, setAudioUrl] = useState<string | null>(null)
  const [loadError, setLoadError] = useState(false)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!shouldLoad || !agentId) {
      setAudioUrl((prev) => {
        if (prev) URL.revokeObjectURL(prev)
        return null
      })
      setLoadError(false)
      setLoading(false)
      return
    }
    let cancelled = false
    setLoadError(false)
    setLoading(true)
    ;(async () => {
      try {
        const res = await api.get(`/agents/${agentId}/voice-preview`, { responseType: 'blob' })
        if (cancelled) return
        const u = URL.createObjectURL(res.data as Blob)
        setAudioUrl((prev) => {
          if (prev) URL.revokeObjectURL(prev)
          return u
        })
      } catch {
        if (!cancelled) setLoadError(true)
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
      setAudioUrl((prev) => {
        if (prev) URL.revokeObjectURL(prev)
        return null
      })
      setLoading(false)
    }
  }, [agentId, shouldLoad])

  return { audioUrl, loadError, loading }
}

/** Prévia de voz guardada no servidor — texto: Olá, sou a agente [nome]... */
function AgentVoicePreviewBlock({
  isCreate,
  agentId,
  voicePreviewAvailable,
  compact,
}: {
  isCreate: boolean
  agentId?: string
  voicePreviewAvailable: boolean
  compact?: boolean
}) {
  const shouldLoad = Boolean(!isCreate && agentId && voicePreviewAvailable)
  const { audioUrl, loadError, loading } = useAgentVoicePreviewUrl(agentId, shouldLoad)

  if (compact) {
    if (!shouldLoad) return null
    if (audioUrl) {
      return <audio key={audioUrl} controls className="h-8 w-full max-w-full" src={audioUrl} preload="metadata" />
    }
    if (loadError) {
      return <p className="text-[11px] text-muted-foreground">Prévia indisponível.</p>
    }
    if (loading) {
      return <p className="text-[11px] text-muted-foreground">A carregar…</p>
    }
    return null
  }

  return (
    <div className="space-y-2 rounded-md border border-border bg-muted/30 p-3">
      <Label className="text-muted-foreground">Amostra da voz</Label>
      {isCreate ? (
        <p className="text-xs text-muted-foreground">
          Guarda o agente para gerar o áudio de exemplo (cerca de 10 segundos, com o nome do agente).
        </p>
      ) : voicePreviewAvailable && audioUrl ? (
        <>
          <audio key={audioUrl} controls className="h-9 w-full max-w-md" src={audioUrl} preload="metadata" />
          <p className="text-xs text-muted-foreground">
            Amostra fixa: «Olá, sou a agente [nome]. Como posso te ajudar hoje?» — atualiza ao gravar quando alteras o
            nome ou a voz.
          </p>
        </>
      ) : voicePreviewAvailable && loading ? (
        <p className="text-xs text-muted-foreground">A carregar a amostra…</p>
      ) : (
        <p className="text-xs text-muted-foreground">
          {loadError
            ? 'Não foi possível carregar a amostra.'
            : 'Ainda sem amostra. Grava com TTS ativo e o provedor (OpenAI, OmniVoice, ElevenLabs ou Kokoro) acessível para gerar o ficheiro.'}
        </p>
      )}
      {!isCreate && (
        <p className="text-xs text-muted-foreground/80">
          O leitor usa a última gravação. Se mudares a voz ou o nome, guarda para regenerar a amostra.
        </p>
      )}
    </div>
  )
}

async function fetchAgents(): Promise<AgentRow[]> {
  const res = await api.get<unknown>('/agents')
  const { data } = unwrapEnvelope<AgentRow[]>(res)
  return data
}

export default function Agents() {
  const qc = useQueryClient()
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<AgentRow | null>(null)
  const [testOpen, setTestOpen] = useState(false)
  const [testAgent, setTestAgent] = useState<AgentRow | null>(null)
  const [testMessage, setTestMessage] = useState('Olá, preciso de um orçamento.')
  const [testReply, setTestReply] = useState<string | null>(null)
  const [keyHint, setKeyHint] = useState<string | null>(null)
  const [ttsKeyHint, setTtsKeyHint] = useState<string | null>(null)
  const [elKeyHint, setElKeyHint] = useState<string | null>(null)

  const isCreate = editing === null

  const form = useForm<AgentFormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: '',
      provider: 'gemini',
      model_preset: GEMINI_MODELS[0],
      model_custom: '',
      api_key: '',
      role: '',
      description: '',
      active: true,
      use_for_whatsapp_auto_reply: false,
      voice_reply_enabled: false,
      tts_provider: 'none',
      openai_tts_voice: 'nova',
      openai_tts_model: '',
      openai_tts_api_key: '',
      omnivoice_base_url: '',
      kokoro_base_url: '',
      elevenlabs_api_key: '',
    },
  })

  const provider = form.watch('provider')
  const modelPreset = form.watch('model_preset')
  const voiceReplyEnabled = form.watch('voice_reply_enabled')
  const ttsProvider = form.watch('tts_provider')

  useEffect(() => {
    if (!formOpen) return
    if (isCreate) {
      form.reset({
        name: '',
        provider: 'gemini',
        model_preset: GEMINI_MODELS[0],
        model_custom: '',
        api_key: '',
        role: '',
        description: '',
        active: true,
        use_for_whatsapp_auto_reply: false,
        voice_reply_enabled: false,
        tts_provider: 'none',
        openai_tts_voice: 'nova',
        openai_tts_model: '',
        openai_tts_api_key: '',
        omnivoice_base_url: '',
        kokoro_base_url: '',
        elevenlabs_api_key: '',
      })
      setKeyHint(null)
      setTtsKeyHint(null)
      setElKeyHint(null)
      return
    }
    const { preset, custom } = resolveModelPreset(editing.model, editing.provider as 'gemini' | 'openai')
    const voiceOn = editing.voice_reply_enabled ?? false
    const allowedTts: TTSProviderValue[] = ['none', 'openai_tts', 'omnivoice', 'elevenlabs', 'kokoro']
    let tts = (editing.tts_provider || 'none') as TTSProviderValue
    if (!allowedTts.includes(tts)) {
      tts = 'openai_tts'
    }
    if (voiceOn && tts === 'none') {
      tts = 'openai_tts'
    }
    form.reset({
      name: editing.name,
      provider: editing.provider as 'gemini' | 'openai',
      model_preset: preset,
      model_custom: custom,
      api_key: '',
      role: editing.role ?? '',
      description: editing.description ?? '',
      active: editing.active,
      use_for_whatsapp_auto_reply: editing.use_for_whatsapp_auto_reply,
      voice_reply_enabled: voiceOn,
      tts_provider: voiceOn ? tts : 'none',
      openai_tts_voice:
        editing.openai_tts_voice && editing.openai_tts_voice.trim() !== ''
          ? editing.openai_tts_voice.trim()
          : defaultVoiceForTTSProvider(voiceOn ? tts : 'none'),
      openai_tts_model: editing.openai_tts_model?.trim() ?? '',
      openai_tts_api_key: '',
      omnivoice_base_url: editing.omnivoice_base_url ?? '',
      kokoro_base_url: editing.kokoro_base_url ?? '',
      elevenlabs_api_key: '',
    })
    setKeyHint(null)
    setTtsKeyHint(null)
    setElKeyHint(null)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- reset só quando abre/fecha ou muda edição
  }, [formOpen, isCreate, editing])

  const { data = [], isLoading } = useQuery({
    queryKey: ['agents'],
    queryFn: fetchAgents,
  })

  const createMut = useMutation({
    mutationFn: async (values: AgentFormValues) => {
      const model =
        values.model_preset === CUSTOM_MODEL
          ? values.model_custom.trim()
          : values.model_preset
      const ttsProv = values.voice_reply_enabled ? values.tts_provider : 'none'
      const res = await api.post<unknown>('/agents', {
        name: values.name.trim(),
        provider: values.provider,
        model,
        api_key: values.api_key.trim(),
        role: values.role.trim(),
        description: values.description.trim(),
        active: values.active,
        use_for_whatsapp_auto_reply: values.use_for_whatsapp_auto_reply,
        voice_reply_enabled: values.voice_reply_enabled && ttsProv !== 'none',
        tts_provider: ttsProv,
        openai_tts_voice: values.openai_tts_voice.trim(),
        openai_tts_model: values.openai_tts_model.trim(),
        openai_tts_api_key: values.openai_tts_api_key.trim(),
        omnivoice_base_url: values.omnivoice_base_url.trim(),
        kokoro_base_url: values.kokoro_base_url.trim(),
        elevenlabs_api_key: values.elevenlabs_api_key.trim(),
      })
      return unwrapEnvelope<AgentRow>(res).data
    },
    onSuccess: () => {
      toast.success('Agente criado.')
      qc.invalidateQueries({ queryKey: ['agents'] })
      setFormOpen(false)
    },
    onError: (err: unknown) => {
      if (err instanceof ApiEnvelopeError) toast.error(err.message)
      else toast.error('Falha ao criar agente.')
    },
  })

  const patchMut = useMutation({
    mutationFn: async ({
      id,
      values,
    }: {
      id: string
      values: AgentFormValues
    }) => {
      const model =
        values.model_preset === CUSTOM_MODEL
          ? values.model_custom.trim()
          : values.model_preset
      const ttsProv = values.voice_reply_enabled ? values.tts_provider : 'none'
      const body: Record<string, unknown> = {
        name: values.name.trim(),
        provider: values.provider,
        model,
        role: values.role.trim(),
        description: values.description.trim(),
        active: values.active,
        use_for_whatsapp_auto_reply: values.use_for_whatsapp_auto_reply,
        voice_reply_enabled: values.voice_reply_enabled && ttsProv !== 'none',
        tts_provider: ttsProv,
        openai_tts_voice: values.openai_tts_voice.trim(),
        openai_tts_model: values.openai_tts_model.trim(),
        omnivoice_base_url: values.omnivoice_base_url.trim(),
        kokoro_base_url: values.kokoro_base_url.trim(),
      }
      if (values.api_key.trim()) {
        body.api_key = values.api_key.trim()
      }
      if (values.openai_tts_api_key.trim()) {
        body.openai_tts_api_key = values.openai_tts_api_key.trim()
      }
      if (values.elevenlabs_api_key.trim()) {
        body.elevenlabs_api_key = values.elevenlabs_api_key.trim()
      }
      const res = await api.patch<unknown>(`/agents/${id}`, body)
      return unwrapEnvelope<AgentRow>(res).data
    },
    onSuccess: () => {
      toast.success('Agente atualizado.')
      qc.invalidateQueries({ queryKey: ['agents'] })
      setFormOpen(false)
    },
    onError: (err: unknown) => {
      if (err instanceof ApiEnvelopeError) toast.error(err.message)
      else toast.error('Falha ao atualizar.')
    },
  })

  const deleteMut = useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/agents/${id}`)
    },
    onSuccess: () => {
      toast.success('Agente eliminado.')
      qc.invalidateQueries({ queryKey: ['agents'] })
    },
    onError: (err: unknown) => {
      if (err instanceof ApiEnvelopeError) toast.error(err.message)
      else toast.error('Falha ao eliminar.')
    },
  })

  const testMut = useMutation({
    mutationFn: async ({ id, message }: { id: string; message: string }) => {
      const res = await api.post<unknown>(`/agents/${id}/test`, { message })
      return unwrapEnvelope<{ reply: string }>(res).data.reply
    },
    onSuccess: (reply) => {
      setTestReply(reply)
      toast.success('Resposta recebida.')
    },
    onError: (err: unknown) => {
      setTestReply(null)
      if (err instanceof ApiEnvelopeError) toast.error(err.message)
      else toast.error('Teste falhou.')
    },
  })

  function openCreate() {
    setEditing(null)
    setFormOpen(true)
  }

  function openEdit(a: AgentRow) {
    setEditing(a)
    setFormOpen(true)
  }

  function onSubmit(values: AgentFormValues) {
    if (isCreate) {
      if (!values.api_key.trim()) {
        toast.error('API key obrigatória ao criar.')
        return
      }
      if (
        values.voice_reply_enabled &&
        values.tts_provider === 'openai_tts' &&
        values.provider === 'gemini' &&
        !values.openai_tts_api_key.trim()
      ) {
        toast.error('Para TTS OpenAI com LLM Gemini, indica a API key OpenAI (só TTS).')
        return
      }
      if (values.voice_reply_enabled && values.tts_provider === 'elevenlabs' && !values.elevenlabs_api_key.trim()) {
        toast.error('Para ElevenLabs, indica a API key (xi-api-key).')
        return
      }
      createMut.mutate(values)
    } else if (editing) {
      if (
        values.voice_reply_enabled &&
        values.tts_provider === 'openai_tts' &&
        values.provider === 'gemini' &&
        !values.openai_tts_api_key.trim() &&
        !editing.has_openai_tts_api_key
      ) {
        toast.error('Indica a API key OpenAI para TTS ou desativa a voz.')
        return
      }
      if (
        values.voice_reply_enabled &&
        values.tts_provider === 'elevenlabs' &&
        !values.elevenlabs_api_key.trim() &&
        !editing.has_elevenlabs_api_key
      ) {
        toast.error('Indica a API key ElevenLabs ou desativa a voz.')
        return
      }
      patchMut.mutate({ id: editing.id, values })
    }
  }

  const docLinks = useMemo(
    () => ({
      gemini: 'https://aistudio.google.com/apikey',
      openai: 'https://platform.openai.com/docs/models',
    }),
    []
  )

  return (
    <div className="p-6 h-full overflow-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Agentes IA</h1>
          <p className="text-sm text-text-muted">
            Modelo, chave e personalidade por workspace. Um agente pode ser usado na auto-resposta WhatsApp.
          </p>
        </div>
        <Button className="bg-primary" type="button" onClick={openCreate}>
          <Bot className="size-4" />
          Novo agente
        </Button>
      </div>

      <Dialog open={formOpen} onOpenChange={setFormOpen}>
        <DialogContent className="bg-card border-border max-w-lg max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{isCreate ? 'Novo agente' : 'Editar agente'}</DialogTitle>
            <DialogDescription>
              A chave API é encriptada no servidor (
              <code className="text-xs">APP_ENCRYPTION_KEY</code>). Nunca é devolvida na API — só indicador e últimos
              dígitos.
            </DialogDescription>
          </DialogHeader>
          <form className="space-y-4" onSubmit={form.handleSubmit(onSubmit)}>
            <div className="flex flex-wrap gap-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="text-xs"
                onClick={() => {
                  form.setValue('name', INFINITI_EXAMPLE.name)
                  form.setValue('role', INFINITI_EXAMPLE.role)
                  form.setValue('description', INFINITI_EXAMPLE.description)
                  toast.message('Exemplo Infiniti Engenharia aplicado — edita à vontade.')
                }}
              >
                Preencher exemplo (persona)
              </Button>
            </div>
            <div className="space-y-2">
              <Label htmlFor="ag-name">Nome do bot</Label>
              <Input id="ag-name" className="bg-background" {...form.register('name')} />
              {form.formState.errors.name && (
                <p className="text-xs text-destructive">{form.formState.errors.name.message}</p>
              )}
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <Label>Fornecedor</Label>
                <Controller
                  control={form.control}
                  name="provider"
                  render={({ field }) => (
                    <Select
                      value={field.value}
                      onValueChange={(v) => {
                        field.onChange(v)
                        const first = modelsForProvider(v as 'gemini' | 'openai')[0]
                        form.setValue('model_preset', first)
                        form.setValue('model_custom', '')
                      }}
                    >
                      <SelectTrigger className="bg-background">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="gemini">Gemini</SelectItem>
                        <SelectItem value="openai">OpenAI</SelectItem>
                      </SelectContent>
                    </Select>
                  )}
                />
                <p className="text-xs text-text-muted">
                  Docs:{' '}
                  <a
                    href={docLinks[provider]}
                    target="_blank"
                    rel="noreferrer"
                    className="text-primary underline"
                  >
                    {provider === 'gemini' ? 'Google AI Studio' : 'OpenAI models'}
                  </a>
                </p>
              </div>
              <div className="space-y-2">
                <Label>Modelo</Label>
                <Controller
                  control={form.control}
                  name="model_preset"
                  render={({ field }) => (
                    <Select value={field.value} onValueChange={field.onChange}>
                      <SelectTrigger className="bg-background">
                        <SelectValue placeholder="Modelo" />
                      </SelectTrigger>
                      <SelectContent>
                        {modelsForProvider(provider).map((m) => (
                          <SelectItem key={m} value={m}>
                            {m}
                          </SelectItem>
                        ))}
                        <SelectItem value={CUSTOM_MODEL}>Personalizado…</SelectItem>
                      </SelectContent>
                    </Select>
                  )}
                />
              </div>
            </div>
            {modelPreset === CUSTOM_MODEL && (
              <div className="space-y-2">
                <Label htmlFor="ag-model-custom">Modelo (texto)</Label>
                <Input
                  id="ag-model-custom"
                  className="bg-background"
                  placeholder="ex. gemini-2.5-flash-preview-09-2025"
                  {...form.register('model_custom')}
                />
                {form.formState.errors.model_custom && (
                  <p className="text-xs text-destructive">{form.formState.errors.model_custom.message}</p>
                )}
              </div>
            )}
            <div className="space-y-2">
              <Label htmlFor="ag-key">API key {isCreate ? '' : '(deixa vazio para manter)'}</Label>
              <Input
                id="ag-key"
                type="password"
                autoComplete="off"
                className="bg-background"
                {...form.register('api_key', {
                  onChange: (e) => setKeyHint(hintFromApiKey(e.target.value)),
                })}
              />
              {keyHint && <p className="text-xs text-muted-foreground">{keyHint}</p>}
            </div>
            <div className="space-y-2">
              <Label htmlFor="ag-role">Função / papel</Label>
              <Input id="ag-role" className="bg-background" {...form.register('role')} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="ag-desc">Descrição / contexto</Label>
              <Textarea id="ag-desc" className="bg-background min-h-[100px]" {...form.register('description')} />
            </div>
            <div className="flex items-center justify-between gap-4">
              <div className="flex items-center gap-2">
                <Controller
                  control={form.control}
                  name="active"
                  render={({ field }) => (
                    <Switch checked={field.value} onCheckedChange={field.onChange} id="ag-active" />
                  )}
                />
                <Label htmlFor="ag-active">Ativo</Label>
              </div>
              <div className="flex items-center gap-2">
                <Controller
                  control={form.control}
                  name="use_for_whatsapp_auto_reply"
                  render={({ field }) => (
                    <Switch checked={field.value} onCheckedChange={field.onChange} id="ag-wa" />
                  )}
                />
                <Label htmlFor="ag-wa" className="text-sm">
                  Usar na auto-resposta WhatsApp
                </Label>
              </div>
            </div>
            <div className="rounded-lg border border-border bg-muted/20 p-3 space-y-3">
              <p className="text-sm font-medium text-text-primary">Resposta em voz (WhatsApp)</p>
              <p className="text-xs text-text-muted">
                Em vez de texto, o agente envia uma mensagem de voz (TTS) quando um contacto escreve no WhatsApp
                (auto-resposta). Requer <code className="text-[10px]">PUBLIC_MEDIA_BASE_URL</code> acessível pela
                Evolution (GET ao ficheiro antes do envio).
              </p>
              <ul className="text-xs text-text-muted list-disc pl-4 space-y-1">
                <li>
                  <strong className="font-medium text-text-primary">OmniVoice:</strong>{' '}
                  <code className="text-[10px]">npm run omnivoice:server</code> (porta 8000);{' '}
                  <code className="text-[10px]">OMNIVOICE_DEFAULT_BASE_URL</code> ou URL no agente; em Docker use{' '}
                  <code className="text-[10px]">http://host.docker.internal:8000</code>
                </li>
                <li>
                  <strong className="font-medium text-text-primary">Kokoro (local):</strong>{' '}
                  <code className="text-[10px]">npm run kokoro:server</code> ou Docker;{' '}
                  <code className="text-[10px]">KOKORO_DEFAULT_BASE_URL</code> ou URL no agente.
                </li>
                <li>
                  <code className="text-[10px]">PUBLIC_MEDIA_BASE_URL</code> acessível ao Evolution (GET ao ficheiro antes do envio).
                </li>
              </ul>
              <div className="flex items-center gap-2">
                <Controller
                  control={form.control}
                  name="voice_reply_enabled"
                  render={({ field }) => (
                    <Switch
                      checked={field.value}
                      onCheckedChange={(v) => {
                        field.onChange(v)
                        if (!v) {
                          form.setValue('tts_provider', 'none')
                        } else if (form.getValues('tts_provider') === 'none') {
                          form.setValue('tts_provider', 'omnivoice')
                          form.setValue('openai_tts_voice', defaultVoiceForTTSProvider('omnivoice'))
                        }
                      }}
                      id="ag-voice"
                    />
                  )}
                />
                <Label htmlFor="ag-voice">Responder em áudio (TTS)</Label>
              </div>
              {voiceReplyEnabled && (
                <>
                  <div className="space-y-2">
                    <Label>Provedor de voz</Label>
                    <Controller
                      control={form.control}
                      name="tts_provider"
                      render={({ field }) => (
                        <Select
                          value={field.value}
                          onValueChange={(v) => {
                            field.onChange(v)
                            form.setValue('openai_tts_voice', defaultVoiceForTTSProvider(v as TTSProviderValue))
                          }}
                        >
                          <SelectTrigger className="bg-background">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="openai_tts">OpenAI TTS (cloud)</SelectItem>
                            <SelectItem value="omnivoice">OmniVoice (local)</SelectItem>
                            <SelectItem value="elevenlabs">ElevenLabs (cloud, pago por carácter)</SelectItem>
                            <SelectItem value="kokoro">Kokoro (local, API compatível OpenAI)</SelectItem>
                          </SelectContent>
                        </Select>
                      )}
                    />
                  </div>
                  {ttsProvider === 'openai_tts' && (
                    <>
                      <div className="space-y-2">
                        <Label>Voz</Label>
                        <Controller
                          control={form.control}
                          name="openai_tts_voice"
                          render={({ field }) => {
                            const hasPreset = OPENAI_TTS_VOICES.some((o) => o.value === field.value)
                            return (
                              <Select value={field.value} onValueChange={field.onChange}>
                                <SelectTrigger className="bg-background">
                                  <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                  {!hasPreset && field.value ? (
                                    <SelectItem value={field.value}>Guardado: {field.value}</SelectItem>
                                  ) : null}
                                  {OPENAI_TTS_VOICES.map((o) => (
                                    <SelectItem key={o.value} value={o.value}>
                                      {o.label} ({o.value})
                                    </SelectItem>
                                  ))}
                                </SelectContent>
                              </Select>
                            )
                          }}
                        />
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="ag-openai-tts-model">Modelo TTS OpenAI (opcional)</Label>
                        <Input
                          id="ag-openai-tts-model"
                          className="bg-background"
                          placeholder="vazio = defeito do servidor (ex. gpt-4o-mini-tts)"
                          {...form.register('openai_tts_model')}
                        />
                        <p className="text-xs text-text-muted">
                          Opcional: sobrepõe <code className="text-[10px]">OPENAI_TTS_MODEL</code> no servidor para este agente.
                        </p>
                      </div>
                      {provider === 'openai' ? (
                        <p className="text-xs text-text-muted">
                          Usa a mesma API key do agente (OpenAI) para sintetizar voz, a menos que cries uma chave
                          dedicada abaixo.
                        </p>
                      ) : null}
                      {(provider === 'gemini' || provider === 'openai') && (
                        <div className="space-y-2">
                          <Label htmlFor="ag-tts-key">
                            API key OpenAI (só TTS)
                            {provider === 'gemini' ? ' — obrigatória (criação)' : ' — opcional'}
                          </Label>
                          <Input
                            id="ag-tts-key"
                            type="password"
                            autoComplete="off"
                            className="bg-background"
                            placeholder={isCreate ? '' : '(deixa vazio para manter)'}
                            {...form.register('openai_tts_api_key', {
                              onChange: (e) => setTtsKeyHint(hintFromApiKey(e.target.value)),
                            })}
                          />
                          {ttsKeyHint && (
                            <p className="text-xs text-muted-foreground">{ttsKeyHint}</p>
                          )}
                          {provider === 'gemini' && !isCreate && editing?.has_openai_tts_api_key && (
                            <p className="text-xs text-text-muted">
                              Chave TTS atual: …{editing.openai_tts_api_key_last4 || '****'}
                            </p>
                          )}
                          {provider === 'openai' && !isCreate && editing?.has_openai_tts_api_key && (
                            <p className="text-xs text-text-muted">
                              Chave TTS dedicada: …{editing.openai_tts_api_key_last4 || '****'}
                            </p>
                          )}
                        </div>
                      )}
                    </>
                  )}
                  {ttsProvider === 'elevenlabs' && (
                    <div className="space-y-3">
                      <p className="text-xs text-text-muted">
                        Custo por carácter na conta ElevenLabs. Documentação:{' '}
                        <a
                          href="https://elevenlabs.io/docs"
                          target="_blank"
                          rel="noreferrer"
                          className="text-primary underline"
                        >
                          elevenlabs.io/docs
                        </a>
                        {' · '}
                        <a
                          href="https://elevenlabs.io/docs/api-reference/text-to-speech/convert"
                          target="_blank"
                          rel="noreferrer"
                          className="text-primary underline"
                        >
                          Text-to-speech
                        </a>
                      </p>
                      <div className="space-y-2">
                        <Label>Voz (voice_id)</Label>
                        <Controller
                          control={form.control}
                          name="openai_tts_voice"
                          render={({ field }) => {
                            const trimmed = (field.value || '').trim()
                            const known = ELEVENLABS_FEMALE_VOICES.some((x) => x.value === trimmed)
                            const selectVal = known ? trimmed : VOICE_CUSTOM
                            return (
                              <>
                                <Select
                                  value={selectVal}
                                  onValueChange={(v) => {
                                    if (v === VOICE_CUSTOM) return
                                    field.onChange(v)
                                  }}
                                >
                                  <SelectTrigger className="bg-background">
                                    <SelectValue placeholder="Preset ou personalizado" />
                                  </SelectTrigger>
                                  <SelectContent>
                                    {ELEVENLABS_FEMALE_VOICES.map((o) => (
                                      <SelectItem key={o.value} value={o.value}>
                                        {o.label}
                                      </SelectItem>
                                    ))}
                                    <SelectItem value={VOICE_CUSTOM}>ID personalizado (clone / biblioteca)</SelectItem>
                                  </SelectContent>
                                </Select>
                                {(!known || selectVal === VOICE_CUSTOM) && (
                                  <Input
                                    className="bg-background"
                                    placeholder="Cole o voice_id (ex. da Voice Library)"
                                    value={field.value}
                                    onChange={field.onChange}
                                    onBlur={field.onBlur}
                                    name={field.name}
                                    ref={field.ref}
                                  />
                                )}
                              </>
                            )
                          }}
                        />
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="ag-el-key">
                          API key ElevenLabs (<code className="text-[10px]">xi-api-key</code>)
                          {isCreate ? ' — obrigatória' : ' — deixa vazio para manter'}
                        </Label>
                        <Input
                          id="ag-el-key"
                          type="password"
                          autoComplete="off"
                          className="bg-background"
                          placeholder={isCreate ? '' : '(deixa vazio para manter)'}
                          {...form.register('elevenlabs_api_key', {
                            onChange: (e) => setElKeyHint(hintFromApiKey(e.target.value)),
                          })}
                        />
                        {elKeyHint && <p className="text-xs text-muted-foreground">{elKeyHint}</p>}
                        {!isCreate && editing?.has_elevenlabs_api_key && (
                          <p className="text-xs text-text-muted">
                            Chave atual: …{editing.elevenlabs_api_key_last4 || '****'}
                          </p>
                        )}
                      </div>
                    </div>
                  )}
                  {ttsProvider === 'kokoro' && (
                    <div className="space-y-3">
                      <div className="space-y-2">
                        <Label htmlFor="ag-kokoro-url">URL base do Kokoro (OpenAI-compat)</Label>
                        <Input
                          id="ag-kokoro-url"
                          className="bg-background"
                          placeholder="http://127.0.0.1:8880"
                          {...form.register('kokoro_base_url')}
                        />
                        <p className="text-xs text-text-muted">
                          Servidor tipo Kokoro-FastAPI com{' '}
                          <code className="text-[10px]">POST /v1/audio/speech</code>. Vazio se definires{' '}
                          <code className="text-[10px]">KOKORO_DEFAULT_BASE_URL</code> na API.
                        </p>
                      </div>
                      <div className="space-y-2">
                        <Label>Voz Kokoro</Label>
                        <Controller
                          control={form.control}
                          name="openai_tts_voice"
                          render={({ field }) => {
                            const trimmed = (field.value || '').trim()
                            const known = KOKORO_VOICE_PRESETS.some((x) => x.value === trimmed)
                            const selectVal = known ? trimmed : VOICE_CUSTOM
                            return (
                              <>
                                <Select
                                  value={selectVal}
                                  onValueChange={(v) => {
                                    if (v === VOICE_CUSTOM) return
                                    field.onChange(v)
                                  }}
                                >
                                  <SelectTrigger className="bg-background">
                                    <SelectValue placeholder="Preset ou ID livre" />
                                  </SelectTrigger>
                                  <SelectContent>
                                    {KOKORO_VOICE_PRESETS.map((o) => (
                                      <SelectItem key={o.value} value={o.value}>
                                        {o.label}
                                      </SelectItem>
                                    ))}
                                    <SelectItem value={VOICE_CUSTOM}>Outro ID (VOICES.md)</SelectItem>
                                  </SelectContent>
                                </Select>
                                {(!known || selectVal === VOICE_CUSTOM) && (
                                  <Input
                                    className="bg-background"
                                    placeholder="ex. pf_dora, af_heart"
                                    value={field.value}
                                    onChange={field.onChange}
                                    onBlur={field.onBlur}
                                    name={field.name}
                                    ref={field.ref}
                                  />
                                )}
                              </>
                            )
                          }}
                        />
                        <p className="text-xs text-text-muted">
                          Vozes oficiais:{' '}
                          <a
                            href="https://huggingface.co/hexgrad/Kokoro-82M/blob/main/VOICES.md"
                            target="_blank"
                            rel="noreferrer"
                            className="text-primary underline"
                          >
                            Kokoro VOICES.md
                          </a>
                        </p>
                      </div>
                    </div>
                  )}
                  {ttsProvider === 'omnivoice' && (
                    <div className="space-y-2">
                      <Label htmlFor="ag-omni">URL base do OmniVoice</Label>
                      <Input
                        id="ag-omni"
                        className="bg-background"
                        placeholder="http://127.0.0.1:8000"
                        {...form.register('omnivoice_base_url')}
                      />
                      <p className="text-xs text-text-muted">
                        Servidor com endpoint compatível <code className="text-[10px]">POST /v1/audio/speech</code>{' '}
                        (OpenAI). Podes deixar vazio se definires <code className="text-[10px]">OMNIVOICE_DEFAULT_BASE_URL</code>{' '}
                        no <code className="text-[10px]">.env</code> da API. Acessível a partir do processo da API (ex.{' '}
                        <code className="text-[10px]">http://host.docker.internal:PORT</code>).
                      </p>
                      {form.formState.errors.omnivoice_base_url && (
                        <p className="text-xs text-destructive">
                          {form.formState.errors.omnivoice_base_url.message}
                        </p>
                      )}
                      <div className="space-y-2">
                        <Label>Voz OmniVoice</Label>
                        <Controller
                          control={form.control}
                          name="openai_tts_voice"
                          render={({ field }) => (
                            <Select value={field.value} onValueChange={field.onChange}>
                              <SelectTrigger className="bg-background">
                                <SelectValue />
                              </SelectTrigger>
                              <SelectContent>
                                <SelectItem value="clone:atendimento_br">
                                  Clone — atendimento BR (o teu áudio em voice-profiles)
                                </SelectItem>
                                <SelectItem value="nova">Design — estilo nova (female)</SelectItem>
                                <SelectItem value="shimmer">Design — estilo shimmer (female)</SelectItem>
                              </SelectContent>
                            </Select>
                          )}
                        />
                        <p className="text-xs text-text-muted">
                          O modo clone usa o teu{' '}
                          <code className="text-[10px]">ref_audio.wav</code> + transcrição em{' '}
                          <code className="text-[10px]">meta.json</code> — sotaque brasileiro vem do áudio de referência,
                          não do modo design (<code className="text-[10px]">portuguese accent</code> soa mais europeu).
                        </p>
                        {form.formState.errors.openai_tts_voice && (
                          <p className="text-xs text-destructive">
                            {form.formState.errors.openai_tts_voice.message}
                          </p>
                        )}
                      </div>
                    </div>
                  )}
                  {voiceReplyEnabled && ttsProvider !== 'none' && (
                    <AgentVoicePreviewBlock
                      isCreate={isCreate}
                      agentId={editing?.id}
                      voicePreviewAvailable={editing?.voice_preview_available === true}
                    />
                  )}
                </>
              )}
            </div>
            <DialogFooter className="gap-2 sm:gap-0">
              <Button type="button" variant="outline" onClick={() => setFormOpen(false)}>
                Cancelar
              </Button>
              <Button type="submit" className="bg-primary" disabled={createMut.isPending || patchMut.isPending}>
                Guardar
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog
        open={testOpen}
        onOpenChange={(o) => {
          setTestOpen(o)
          if (!o) {
            setTestReply(null)
            setTestAgent(null)
          }
        }}
      >
        <DialogContent className="bg-card border-border max-w-md">
          <DialogHeader>
            <DialogTitle>Testar agente</DialogTitle>
            <DialogDescription>
              Envia uma mensagem de teste ao LLM configurado (consome quota da API). Só devolve texto — não simula
              TTS nem envio de áudio pela WhatsApp; a voz na auto-resposta real depende do webhook e do provedor TTS
              escolhido no agente.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label>Mensagem</Label>
            <Textarea
              className="bg-background"
              value={testMessage}
              onChange={(e) => setTestMessage(e.target.value)}
              rows={3}
            />
          </div>
          {testReply && (
            <div className="rounded-md border border-border bg-muted/30 p-3 text-sm whitespace-pre-wrap">
              {testReply}
            </div>
          )}
          <DialogFooter>
            <Button
              type="button"
              className="bg-primary"
              disabled={!testAgent || testMut.isPending}
              onClick={() => {
                if (!testAgent) return
                setTestReply(null)
                testMut.mutate({ id: testAgent.id, message: testMessage.trim() || 'Olá' })
              }}
            >
              Enviar teste
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {isLoading ? (
        <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-4">
          <Skeleton className="h-40 rounded-xl" />
          <Skeleton className="h-40 rounded-xl" />
        </div>
      ) : data.length === 0 ? (
        <Card className="bg-card border-border">
          <CardContent className="py-10 text-center text-text-muted text-sm">
            Ainda não há agentes. Cria um para ligar um modelo à auto-resposta WhatsApp (com{' '}
            <code className="text-xs">AUTO_REPLY_ENABLED</code> no servidor).
          </CardContent>
        </Card>
      ) : (
        <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-4">
          {data.map((a) => (
            <Card key={a.id} className="bg-card border-border">
              <CardHeader className="flex flex-row items-start justify-between space-y-0 pb-2">
                <CardTitle className="text-base font-semibold flex items-center gap-2">
                  <Sparkles className="size-4 text-primary" />
                  {a.name}
                </CardTitle>
                <Badge variant={a.active ? 'default' : 'secondary'}>{a.active ? 'Ativo' : 'Inativo'}</Badge>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="flex flex-wrap gap-2">
                  <Badge variant="outline">{a.provider}</Badge>
                  <Badge variant="outline">{a.model}</Badge>
                  {a.use_for_whatsapp_auto_reply && (
                    <Badge className="bg-primary/15 text-primary border-primary/30">WhatsApp auto</Badge>
                  )}
                  {a.voice_reply_enabled && a.tts_provider && a.tts_provider !== 'none' && (
                    <Badge variant="secondary" className="border-border">
                      Voz ({ttsProviderBadgeLabel(a.tts_provider)})
                    </Badge>
                  )}
                </div>
                {a.has_api_key ? (
                  <p className="text-xs text-text-muted">Chave: …{a.api_key_last4 || '****'}</p>
                ) : (
                  <p className="text-xs text-destructive">Sem chave guardada</p>
                )}
                {a.role && <p className="text-xs text-text-muted line-clamp-2">{a.role}</p>}
                {a.voice_reply_enabled && a.voice_preview_available && (
                  <div className="space-y-1 rounded-md border border-border/60 bg-muted/20 p-2">
                    <p className="text-[11px] font-medium text-muted-foreground">Amostra da voz</p>
                    <AgentVoicePreviewBlock
                      isCreate={false}
                      agentId={a.id}
                      voicePreviewAvailable
                      compact
                    />
                  </div>
                )}
                <div className="flex flex-wrap gap-2 pt-1">
                  <Button variant="secondary" size="sm" className="gap-1" type="button" onClick={() => openEdit(a)}>
                    <Pencil className="size-3.5" />
                    Editar
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    className="gap-1"
                    type="button"
                    disabled={!a.active}
                    onClick={() => {
                      setTestAgent(a)
                      setTestReply(null)
                      setTestOpen(true)
                    }}
                  >
                    <FlaskConical className="size-3.5" />
                    Testar
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="gap-1 text-destructive"
                    type="button"
                    onClick={() => {
                      if (window.confirm(`Eliminar o agente “${a.name}”?`)) {
                        deleteMut.mutate(a.id)
                      }
                    }}
                  >
                    <Trash2 className="size-3.5" />
                    Eliminar
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
