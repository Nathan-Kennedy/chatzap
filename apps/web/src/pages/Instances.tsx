import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { History, Link2, QrCode, RefreshCw, Smartphone, Trash2 } from 'lucide-react'
import { api, unwrapEnvelope } from '@/lib/api'
import { cn } from '@/lib/utils'
import { ApiEnvelopeError } from '@/types/api'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { toast } from 'sonner'

type InstanceRow = {
  id: string
  name: string
  evolution_instance_name?: string
  number: string
  status: 'connected' | 'qr_pending' | 'disconnected'
  messages_today: number
}

async function fetchInstances(): Promise<InstanceRow[]> {
  const res = await api.get<unknown>('/instances')
  const { data } = unwrapEnvelope<InstanceRow[]>(res)
  return data
}

function statusBadge(status: InstanceRow['status']) {
  switch (status) {
    case 'connected':
      return <Badge className="bg-success/20 text-success border-success/30">Conectado</Badge>
    case 'qr_pending':
      return <Badge className="bg-warning/20 text-warning border-warning/30">QR pendente</Badge>
    default:
      return <Badge variant="destructive">Desconectado</Badge>
  }
}

export default function Instances() {
  const qc = useQueryClient()
  const { data = [], isLoading, error } = useQuery({
    queryKey: ['instances'],
    queryFn: fetchInstances,
    refetchInterval: false,
  })

  const [createOpen, setCreateOpen] = useState(false)
  const [evoName, setEvoName] = useState('')
  const [displayName, setDisplayName] = useState('')

  const [importOpen, setImportOpen] = useState(false)
  const [importName, setImportName] = useState('')
  const [importToken, setImportToken] = useState('')
  const [importDisplay, setImportDisplay] = useState('')

  const [qrOpen, setQrOpen] = useState(false)
  const [qrForId, setQrForId] = useState<string | null>(null)
  /** Status da linha no momento em que se abriu o modal (evita toast falso se a lista estava desactualizada). */
  const statusWhenQrModalOpenedRef = useRef<InstanceRow['status'] | null>(null)
  type QrPanel = {
    phase: 'loading' | 'image' | 'pairing_only' | 'already_connected' | 'error'
    src: string | null
    pairing: string | null
    hint: string | null
  }
  const [qrPanel, setQrPanel] = useState<QrPanel>({
    phase: 'loading',
    src: null,
    pairing: null,
    hint: null,
  })

  function isLikelyQrImageSrc(s: string): boolean {
    const t = s.trim()
    if (!t) return false
    if (t.startsWith('data:image/')) return t.length > 80
    if (t.startsWith('http://') || t.startsWith('https://')) return true
    return false
  }

  const [syncHistOpen, setSyncHistOpen] = useState(false)
  const [syncHistInstanceId, setSyncHistInstanceId] = useState<string | null>(null)
  const [syncHistPhone, setSyncHistPhone] = useState('69993378283')

  const [deleteOpen, setDeleteOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<InstanceRow | null>(null)
  const [deleteConfirmText, setDeleteConfirmText] = useState('')

  /** Nome que o utilizador deve escrever (Evolution / técnico; fallback ao nome na tabela). */
  function instanceDeleteConfirmKey(row: InstanceRow | null): string {
    if (!row) return ''
    const tech = row.evolution_instance_name?.trim()
    if (tech) return tech
    return row.name.trim()
  }

  const createMut = useMutation({
    mutationFn: async () => {
      const res = await api.post('/instances', {
        evolution_instance_name: evoName.trim().toLowerCase(),
        display_name: displayName.trim() || evoName.trim(),
      })
      return unwrapEnvelope<InstanceRow>(res).data
    },
    onSuccess: () => {
      toast.success('Instância criada na Evolution')
      setCreateOpen(false)
      setEvoName('')
      setDisplayName('')
      void qc.invalidateQueries({ queryKey: ['instances'] })
    },
    onError: (err: unknown) => {
      if (err instanceof ApiEnvelopeError) toast.error(err.message)
      else toast.error('Falha ao criar instância (Evolution a correr?)')
    },
  })

  const importMut = useMutation({
    mutationFn: async () => {
      const res = await api.post('/instances/import', {
        evolution_instance_name: importName.trim().toLowerCase(),
        evolution_instance_token: importToken.trim(),
        display_name: importDisplay.trim() || importName.trim(),
      })
      return unwrapEnvelope<InstanceRow>(res).data
    },
    onSuccess: () => {
      toast.success('Instância importada e webhook sincronizado')
      setImportOpen(false)
      setImportName('')
      setImportToken('')
      setImportDisplay('')
      void qc.invalidateQueries({ queryKey: ['instances'] })
    },
    onError: (err: unknown) => {
      if (err instanceof ApiEnvelopeError) toast.error(err.message)
      else toast.error('Falha ao importar (nome/token ou Evolution inacessível)')
    },
  })

  const syncWebhookMut = useMutation({
    mutationFn: async (id: string) => {
      const res = await api.post<unknown>(`/instances/${id}/sync-webhook`)
      return unwrapEnvelope<{ ok: boolean; webhook_url: string }>(res).data
    },
    onSuccess: () => {
      toast.success('Webhook atualizado na Evolution')
    },
    onError: (err: unknown) => {
      if (err instanceof ApiEnvelopeError) toast.error(err.message)
      else toast.error('Falha ao sincronizar webhook')
    },
  })

  const deleteMut = useMutation({
    mutationFn: async (id: string) => {
      const res = await api.delete<unknown>(`/instances/${id}`)
      return unwrapEnvelope<{ ok: boolean }>(res).data
    },
    onSuccess: () => {
      toast.success('Instância removida')
      setDeleteOpen(false)
      setDeleteTarget(null)
      setDeleteConfirmText('')
      void qc.invalidateQueries({ queryKey: ['instances'] })
      void qc.invalidateQueries({ queryKey: ['conversations'] })
    },
    onError: (err: unknown) => {
      if (err instanceof ApiEnvelopeError) toast.error(err.message)
      else toast.error('Não foi possível remover a instância')
    },
  })

  const syncChatsMut = useMutation({
    mutationFn: async () => {
      if (!syncHistInstanceId) throw new Error('instância')
      const res = await api.post<unknown>(`/instances/${syncHistInstanceId}/sync-chats`, {
        phone: syncHistPhone.trim(),
      })
      return unwrapEnvelope<{ inserted: number; parsed: number; conversation_id: string }>(res).data
    },
    onSuccess: (d) => {
      if (d.inserted > 0) {
        toast.success(`${d.inserted} mensagens importadas para a conversa`)
      } else {
        toast.message(
          d.parsed > 0
            ? `Nenhuma mensagem nova (${d.parsed} já existiam ou formato vazio)`
            : 'Nenhuma mensagem devolvida pela Evolution (ou endpoint indisponível)'
        )
      }
      setSyncHistOpen(false)
      void qc.invalidateQueries({ queryKey: ['conversations'] })
    },
    onError: (err: unknown) => {
      if (err instanceof ApiEnvelopeError) {
        if (err.code === 'sync_not_supported') {
          toast.message(err.message)
        } else toast.error(err.message)
      } else toast.error('Falha ao sincronizar histórico')
    },
  })

  async function loadQr(id: string) {
    setQrPanel({ phase: 'loading', src: null, pairing: null, hint: null })
    try {
      const res = await api.get<unknown>(`/instances/${id}/qrcode`)
      const { data } = unwrapEnvelope<{
        code: string
        pairing_code?: string
        already_connected?: boolean
        evolution_status?: string
      }>(res)

      if (data.already_connected) {
        setQrPanel({
          phase: 'already_connected',
          src: null,
          pairing: null,
          hint: data.evolution_status ?? 'connected',
        })
        return
      }

      const pairing = data.pairing_code?.trim() || null
      const rawCode = data.code?.trim() || ''
      const src = isLikelyQrImageSrc(rawCode) ? rawCode : null

      if (src) {
        setQrPanel({ phase: 'image', src, pairing, hint: null })
        return
      }
      if (pairing) {
        setQrPanel({ phase: 'pairing_only', src: null, pairing, hint: null })
        return
      }

      setQrPanel({
        phase: 'error',
        src: null,
        pairing: null,
        hint: 'A API não devolveu imagem de QR. Se a instância já está ligada, não precisas de QR.',
      })
    } catch {
      toast.error('Não foi possível obter o QR')
      setQrPanel({ phase: 'error', src: null, pairing: null, hint: null })
    }
  }

  useEffect(() => {
    if (!qrOpen || !qrForId) return
    void loadQr(qrForId)
    const t = setInterval(() => {
      void qc.invalidateQueries({ queryKey: ['instances'] })
    }, 4000)
    return () => clearInterval(t)
  }, [qrOpen, qrForId, qc])

  /**
   * Quando o utilizador escaneia o QR, o refetch traz `connected` — fecha o modal e confirma com toast.
   * Só dispara se ao abrir o modal o status **não** era já `connected` (pareamento real qr_pending/disconnected → connected).
   */
  useEffect(() => {
    if (!qrOpen || !qrForId) return
    const row = data.find((r) => r.id === qrForId)
    if (!row || row.status !== 'connected') return
    if (qrPanel.phase !== 'image' && qrPanel.phase !== 'pairing_only') return
    const openedAs = statusWhenQrModalOpenedRef.current
    if (openedAs === 'connected') return
    const label = row.name?.trim() || 'Instância'
    toast.success(`WhatsApp ligado com sucesso (${label}).`, {
      id: `instance-qr-paired-${qrForId}`,
    })
    setQrOpen(false)
    setQrForId(null)
    setQrPanel({ phase: 'loading', src: null, pairing: null, hint: null })
    statusWhenQrModalOpenedRef.current = null
  }, [qrOpen, qrForId, data, qrPanel.phase])

  useEffect(() => {
    if (error) {
      toast.error(
        'API de instâncias indisponível. Use WHATSAPP_PROVIDER=evolution e Evolution no Docker.'
      )
    }
  }, [error])

  return (
    <div className="p-6 h-full flex flex-col gap-4 min-h-0 overflow-auto">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Instâncias WhatsApp</h1>
          <p className="text-sm text-text-muted">Evolution Go — criar, importar do Manager e parear com QR</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Dialog
            open={importOpen}
            onOpenChange={(o) => {
              setImportOpen(o)
              if (!o) {
                setImportName('')
                setImportToken('')
                setImportDisplay('')
              }
            }}
          >
            <DialogTrigger asChild>
              <Button variant="outline" className="border-border">
                <Link2 className="size-4" />
                Importar existente
              </Button>
            </DialogTrigger>
            <DialogContent className="bg-card border-border sm:max-w-md">
              <DialogHeader>
                <DialogTitle>Importar instância</DialogTitle>
              </DialogHeader>
              <div className="space-y-4 py-2">
                <div className="space-y-1.5">
                  <Label htmlFor="imp-evo">Nome técnico (igual ao Manager)</Label>
                  <Input
                    id="imp-evo"
                    placeholder="ex: minha-instancia"
                    value={importName}
                    onChange={(e) => setImportName(e.target.value)}
                    className="bg-background"
                  />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="imp-tok">Token da instância</Label>
                  <Input
                    id="imp-tok"
                    type="password"
                    autoComplete="off"
                    placeholder="UUID do Manager"
                    value={importToken}
                    onChange={(e) => setImportToken(e.target.value)}
                    className="bg-background"
                  />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="imp-disp">Nome a exibir</Label>
                  <Input
                    id="imp-disp"
                    placeholder="Opcional"
                    value={importDisplay}
                    onChange={(e) => setImportDisplay(e.target.value)}
                    className="bg-background"
                  />
                </div>
                <Button
                  className="w-full"
                  disabled={importMut.isPending || !importName.trim() || !importToken.trim()}
                  onClick={() => importMut.mutate()}
                >
                  {importMut.isPending ? 'A importar…' : 'Importar e configurar webhook'}
                </Button>
              </div>
            </DialogContent>
          </Dialog>

          <Dialog open={createOpen} onOpenChange={setCreateOpen}>
            <DialogTrigger asChild>
              <Button className="bg-primary">
                <Smartphone className="size-4" />
                Nova instância
              </Button>
            </DialogTrigger>
            <DialogContent className="bg-card border-border sm:max-w-md">
              <DialogHeader>
                <DialogTitle>Nova instância</DialogTitle>
              </DialogHeader>
              <div className="space-y-4 py-2">
                <div className="space-y-1.5">
                  <Label htmlFor="evo">Nome técnico (Evolution)</Label>
                  <Input
                    id="evo"
                    placeholder="ex: loja_sp"
                    value={evoName}
                    onChange={(e) => setEvoName(e.target.value)}
                    className="bg-background"
                  />
                  <p className="text-[11px] text-text-muted">
                    Apenas letras minúsculas, números, _ e - (único na Evolution).
                  </p>
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="disp">Nome a exibir</Label>
                  <Input
                    id="disp"
                    placeholder="Loja SP"
                    value={displayName}
                    onChange={(e) => setDisplayName(e.target.value)}
                    className="bg-background"
                  />
                </div>
                <Button
                  className="w-full"
                  disabled={createMut.isPending || !evoName.trim()}
                  onClick={() => createMut.mutate()}
                >
                  {createMut.isPending ? 'A criar…' : 'Criar na Evolution'}
                </Button>
              </div>
            </DialogContent>
          </Dialog>

          <Dialog
            open={syncHistOpen}
            onOpenChange={(o) => {
              setSyncHistOpen(o)
              if (!o) setSyncHistInstanceId(null)
            }}
          >
            <DialogContent className="bg-card border-border sm:max-w-md">
              <DialogHeader>
                <DialogTitle>Sincronizar histórico (Evolution)</DialogTitle>
              </DialogHeader>
              <div className="space-y-4 py-2">
                <p className="text-xs text-text-muted">
                  Chama POST /chat/findMessages na Evolution. Em muitas builds do Evolution Go esta rota
                  devolve 501 — o histórico em tempo real continua a ser o webhook.
                </p>
                <div className="space-y-1.5">
                  <Label htmlFor="sync-phone">Telefone do contacto</Label>
                  <Input
                    id="sync-phone"
                    value={syncHistPhone}
                    onChange={(e) => setSyncHistPhone(e.target.value)}
                    className="bg-background"
                    placeholder="69993378283"
                  />
                </div>
                <Button
                  className="w-full"
                  disabled={syncChatsMut.isPending || !syncHistPhone.trim()}
                  onClick={() => syncChatsMut.mutate()}
                >
                  {syncChatsMut.isPending ? 'A sincronizar…' : 'Importar mensagens'}
                </Button>
              </div>
            </DialogContent>
          </Dialog>

          <Dialog
            open={qrOpen}
            onOpenChange={(o) => {
              setQrOpen(o)
              if (!o) {
                setQrForId(null)
                setQrPanel({ phase: 'loading', src: null, pairing: null, hint: null })
                statusWhenQrModalOpenedRef.current = null
              }
            }}
          >
            <DialogContent className="bg-card border-border sm:max-w-md">
              <DialogHeader>
                <DialogTitle>Conectar WhatsApp</DialogTitle>
              </DialogHeader>
              <div className="flex flex-col items-center gap-4 py-6">
                <div className="rounded-2xl border-2 border-dashed border-border p-4 bg-background min-h-[200px] w-full max-w-[280px] flex items-center justify-center">
                  {qrPanel.phase === 'loading' && (
                    <QrCode className="size-32 text-text-muted mx-auto animate-pulse" />
                  )}
                  {qrPanel.phase === 'image' && qrPanel.src && (
                    <img src={qrPanel.src} alt="QR WhatsApp" className="max-w-[240px] max-h-[240px]" />
                  )}
                  {qrPanel.phase === 'already_connected' && (
                    <p className="text-sm text-center text-success px-2">
                      Esta instância já está ligada ao WhatsApp. Não é necessário QR.
                      {qrPanel.hint ? (
                        <span className="block mt-2 text-text-muted text-xs">Estado: {qrPanel.hint}</span>
                      ) : null}
                    </p>
                  )}
                  {qrPanel.phase === 'pairing_only' && qrPanel.pairing && (
                    <div className="text-center px-2">
                      <p className="text-xs text-text-muted mb-2">Código de pareamento</p>
                      <p className="text-2xl font-mono font-bold tracking-widest break-all">
                        {qrPanel.pairing}
                      </p>
                      <p className="text-xs text-text-muted mt-3">
                        No WhatsApp: Aparelhos ligados → Ligar um aparelho → Parear com código.
                      </p>
                    </div>
                  )}
                  {qrPanel.phase === 'error' && (
                    <p className="text-sm text-center text-destructive px-2">
                      {qrPanel.hint ?? 'Não foi possível carregar o QR.'}
                    </p>
                  )}
                </div>
                {qrPanel.phase === 'image' ? (
                  <p className="text-sm text-text-secondary text-center">
                    Escaneie com o WhatsApp. A lista atualiza automaticamente.
                  </p>
                ) : null}
                {qrPanel.pairing && qrPanel.phase === 'image' ? (
                  <p className="text-xs text-text-muted text-center">
                    Ou use o código: <span className="font-mono">{qrPanel.pairing}</span>
                  </p>
                ) : null}
              </div>
            </DialogContent>
          </Dialog>

          <Dialog
            open={deleteOpen}
            onOpenChange={(o) => {
              setDeleteOpen(o)
              if (!o) {
                setDeleteTarget(null)
                setDeleteConfirmText('')
              }
            }}
          >
            <DialogContent className="bg-card border-border sm:max-w-md">
              <DialogHeader className="space-y-3 pr-8">
                <DialogTitle className="flex items-center gap-2 text-destructive text-xl font-semibold">
                  <Trash2 className="size-6 shrink-0 stroke-[1.75]" aria-hidden />
                  Remover Instância
                </DialogTitle>
              </DialogHeader>
              <div className="space-y-4 py-1">
                <p className="text-sm text-text-secondary leading-relaxed">
                  Você está prestes a remover a instância{' '}
                  <strong className="text-text-primary font-semibold">
                    {instanceDeleteConfirmKey(deleteTarget)}
                  </strong>
                  . Esta ação não pode ser desfeita. Todas as conversas e mensagens deste workspace
                  associadas a esta instância serão eliminadas.
                </p>
                <div className="space-y-2">
                  <Label htmlFor="delete-instance-confirm" className="text-sm text-text-secondary">
                    Digite o nome da instância para confirmar:
                  </Label>
                  <Input
                    id="delete-instance-confirm"
                    autoComplete="off"
                    autoCorrect="off"
                    spellCheck={false}
                    placeholder={instanceDeleteConfirmKey(deleteTarget)}
                    value={deleteConfirmText}
                    onChange={(e) => setDeleteConfirmText(e.target.value)}
                    className="bg-background border-border font-mono text-sm"
                  />
                </div>
              </div>
              <DialogFooter className="gap-2 sm:gap-2">
                <Button
                  type="button"
                  variant="outline"
                  className="border-border"
                  onClick={() => {
                    setDeleteOpen(false)
                    setDeleteTarget(null)
                    setDeleteConfirmText('')
                  }}
                >
                  Cancelar
                </Button>
                <Button
                  type="button"
                  className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                  disabled={
                    deleteMut.isPending ||
                    !deleteTarget ||
                    deleteConfirmText.trim() !== instanceDeleteConfirmKey(deleteTarget)
                  }
                  onClick={() => {
                    if (deleteTarget) deleteMut.mutate(deleteTarget.id)
                  }}
                >
                  {deleteMut.isPending ? 'A remover…' : 'Remover Instância'}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      <div className="rounded-xl border border-border bg-card overflow-hidden">
        {isLoading ? (
          <div className="p-4">
            <Skeleton className="h-24 w-full" />
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow className="border-border hover:bg-transparent">
                <TableHead>Nome</TableHead>
                <TableHead>Número</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Msgs hoje</TableHead>
                <TableHead className="text-right">Ações</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.map((row) => (
                <TableRow key={row.id} className="border-border">
                  <TableCell className="font-medium">{row.name}</TableCell>
                  <TableCell className="text-text-muted">{row.number}</TableCell>
                  <TableCell>{statusBadge(row.status)}</TableCell>
                  <TableCell>{row.messages_today}</TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-1 flex-wrap">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          statusWhenQrModalOpenedRef.current = row.status
                          setQrForId(row.id)
                          setQrOpen(true)
                        }}
                      >
                        QR
                      </Button>
                      <Button
                        variant="secondary"
                        size="sm"
                        disabled={syncWebhookMut.isPending}
                        onClick={() => syncWebhookMut.mutate(row.id)}
                        title="Reconfigura POST /webhook/set na Evolution"
                      >
                        <RefreshCw
                          className={cn(
                            'size-3.5',
                            syncWebhookMut.isPending &&
                              syncWebhookMut.variables === row.id &&
                              'animate-spin',
                          )}
                        />
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        title="Tentar importar histórico (findMessages)"
                        onClick={() => {
                          setSyncHistInstanceId(row.id)
                          setSyncHistOpen(true)
                        }}
                      >
                        <History className="size-3.5" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-destructive hover:text-destructive hover:bg-destructive/10"
                        title="Remover instância"
                        onClick={() => {
                          setDeleteConfirmText('')
                          setDeleteTarget(row)
                          setDeleteOpen(true)
                        }}
                      >
                        <Trash2 className="size-3.5" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </div>
    </div>
  )
}
