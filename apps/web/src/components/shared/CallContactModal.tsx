import { useMutation, useQuery } from '@tanstack/react-query'
import { ExternalLink, Loader2, MessageCircle, Phone } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { api } from '@/lib/api'
import { digitsForDial, telUrl, waMeUrl } from '@/utils/phone'
import { avatarColorClass, initialsFromName } from '@/utils/initials'
import { cn } from '@/lib/utils'
import { ApiEnvelopeError, type ApiSuccessEnvelope } from '@/types/api'

/** Documentação oficial — chamadas iniciadas pela empresa (WABA), não é integração automática aqui. */
const META_CALLING_DOCS =
  'https://developers.facebook.com/docs/whatsapp/cloud-api/calling/business-initiated-calls'

const ELEVENLABS_TWILIO_DOCS = 'https://elevenlabs.io/docs/conversational-ai/integrations/twilio'

type ApiMetaPayload = {
  elevenlabs_outbound_call_enabled?: boolean
}

type ElevenLabsOutboundResponse = {
  success: boolean
  message: string
  conversation_id?: string | null
  call_sid?: string | null
}

type Props = {
  open: boolean
  onOpenChange: (v: boolean) => void
  contactName: string
  /** telefone normalizado ou JID — usado para wa.me / tel */
  phoneOrJid: string
}

export function CallContactModal({ open, onOpenChange, contactName, phoneOrJid }: Props) {
  const wa = waMeUrl(phoneOrJid)
  const tel = telUrl(phoneOrJid)
  const av = avatarColorClass(contactName)
  const ini = initialsFromName(contactName)
  const digits = digitsForDial(phoneOrJid)
  const toE164 = digits.length >= 8 ? `+${digits}` : ''

  const { data: meta } = useQuery({
    queryKey: ['api-meta'],
    queryFn: async () => {
      const res = await api.get<ApiSuccessEnvelope<ApiMetaPayload>>('/meta')
      return res.data.data
    },
    enabled: open,
    staleTime: 60_000,
  })

  const elevenOutbound = Boolean(meta?.elevenlabs_outbound_call_enabled)

  const outboundMut = useMutation({
    mutationFn: async () => {
      const res = await api.post<ApiSuccessEnvelope<ElevenLabsOutboundResponse>>('/elevenlabs/outbound-call', {
        to_number: toE164,
      })
      return res.data.data
    },
    onSuccess: (data) => {
      const extra =
        data.call_sid != null && String(data.call_sid).trim() !== ''
          ? ` (CallSid: ${data.call_sid})`
          : ''
      toast.success(data.message || 'Pedido de ligação enviado.', { description: extra || undefined })
    },
    onError: (e: unknown) => {
      if (e instanceof ApiEnvelopeError) {
        toast.error(e.message)
        return
      }
      toast.error('Falha ao iniciar ligação ElevenLabs.')
    },
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="bg-[#0b141a] border-zinc-700 text-zinc-100 sm:max-w-md overflow-hidden p-0 gap-0">
        <div className="px-6 pt-8 pb-4 flex flex-col items-center text-center">
          <div
            className={cn(
              'w-28 h-28 rounded-full flex items-center justify-center text-3xl font-medium text-white mb-4 ring-4 ring-zinc-600/80',
              av,
            )}
          >
            {ini}
          </div>
          <DialogHeader className="space-y-1 text-center sm:text-center">
            <DialogTitle className="text-xl text-white font-medium">{contactName}</DialogTitle>
            <DialogDescription className="text-zinc-400 text-sm">
              {digitsForDial(phoneOrJid) || phoneOrJid}
            </DialogDescription>
          </DialogHeader>
          <p className="text-xs text-zinc-500 mt-4 max-w-sm leading-relaxed text-left">
            Não existe API pública (nem Evolution/Baileys) que inicie uma{' '}
            <strong className="text-zinc-400">chamada de voz ou vídeo no próprio WhatsApp</strong> do contacto a
            partir desta página, da mesma forma que no telemóvel. Para isso a Meta só oferece a{' '}
            <strong className="text-zinc-400">WhatsApp Business Platform</strong> (conta comercial, consentimento do
            cliente, integração própria ou BSP) — ver documentação oficial abaixo.
          </p>
          <p className="text-xs text-zinc-500 mt-3 max-w-sm leading-relaxed text-left">
            Daqui consegues abrir a <strong className="text-zinc-400">conversa no WhatsApp</strong> no dispositivo ou
            fazer uma <strong className="text-zinc-400">chamada telefónica</strong> (rede móvel/operadora), se tiveres o
            número.
          </p>
        </div>
        <div className="px-4 pb-2 flex flex-col gap-2">
          {wa ? (
            <Button
              type="button"
              className="w-full bg-[#00a884] hover:bg-[#06c998] text-white gap-2 h-12"
              asChild
            >
              <a href={wa} target="_blank" rel="noreferrer">
                <MessageCircle className="size-5" />
                Abrir conversa no WhatsApp
              </a>
            </Button>
          ) : null}
          {tel ? (
            <Button type="button" variant="outline" className="w-full border-zinc-600 text-zinc-200 gap-2 h-11" asChild>
              <a href={tel}>
                <Phone className="size-4" />
                Ligar por telefone (operadora)
              </a>
            </Button>
          ) : null}
          {elevenOutbound && toE164 ? (
            <div className="rounded-md border border-zinc-700/80 bg-zinc-900/40 p-3 space-y-2">
              <p className="text-[11px] text-zinc-400 text-left leading-relaxed">
                Ligação com <strong className="text-zinc-300">agente ElevenLabs</strong> (ConvAI) via Twilio. O número
                tem de incluir o indicativo do país (E.164). Confirma no painel ElevenLabs que o Twilio está associado ao
                agente.
              </p>
              <Button
                type="button"
                className="w-full bg-violet-600 hover:bg-violet-500 text-white gap-2 h-11"
                disabled={outboundMut.isPending}
                onClick={() => outboundMut.mutate()}
              >
                {outboundMut.isPending ? (
                  <Loader2 className="size-4 animate-spin" />
                ) : (
                  <Phone className="size-4" />
                )}
                Ligar com agente ElevenLabs
              </Button>
              <a
                href={ELEVENLABS_TWILIO_DOCS}
                target="_blank"
                rel="noreferrer"
                className="flex items-center justify-center gap-1 text-[10px] text-zinc-500 hover:text-zinc-400"
              >
                Documentação Twilio + ElevenLabs
                <ExternalLink className="size-3 shrink-0 opacity-70" />
              </a>
            </div>
          ) : null}
          <a
            href={META_CALLING_DOCS}
            target="_blank"
            rel="noreferrer"
            className="flex items-center justify-center gap-1.5 text-[11px] text-zinc-500 hover:text-zinc-400 py-2 transition-colors"
          >
            <span>Chamadas via API oficial (empresas, Meta)</span>
            <ExternalLink className="size-3 shrink-0 opacity-70" />
          </a>
          {!wa && !tel ? (
            <p className="text-sm text-zinc-500 text-center py-2">Número não reconhecido para estes atalhos.</p>
          ) : null}
        </div>
        <DialogFooter className="px-4 pb-4 pt-2 sm:justify-center border-t border-zinc-800/80">
          <Button type="button" variant="ghost" className="text-zinc-400" onClick={() => onOpenChange(false)}>
            Fechar
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
