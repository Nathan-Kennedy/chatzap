import { ExternalLink, MessageCircle, Phone } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { digitsForDial, telUrl, waMeUrl } from '@/utils/phone'
import { avatarColorClass, initialsFromName } from '@/utils/initials'
import { cn } from '@/lib/utils'

/** Documentação oficial — chamadas iniciadas pela empresa (WABA), não é integração automática aqui. */
const META_CALLING_DOCS =
  'https://developers.facebook.com/docs/whatsapp/cloud-api/calling/business-initiated-calls'

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
