import { useState } from 'react'
import { Plus } from 'lucide-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, unwrapEnvelope } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { toast } from 'sonner'
import { ApiEnvelopeError } from '@/types/api'

type Campaign = {
  id: string
  name: string
  channel: string
  status: string
  sent: number
  delivered: number
  read: number
}

async function fetchCampaigns(): Promise<Campaign[]> {
  const res = await api.get<unknown>('/campaigns')
  const { data } = unwrapEnvelope<Campaign[]>(res)
  return data
}

export default function Campaigns() {
  const qc = useQueryClient()
  const [sheetOpen, setSheetOpen] = useState(false)
  const [name, setName] = useState('')
  const [channel, setChannel] = useState('whatsapp')

  const { data = [], isLoading } = useQuery({
    queryKey: ['campaigns'],
    queryFn: fetchCampaigns,
  })

  const createMut = useMutation({
    mutationFn: async () => {
      const res = await api.post<unknown>('/campaigns', {
        name: name.trim(),
        channel: channel.trim() || 'whatsapp',
      })
      return unwrapEnvelope<{ id: string }>(res).data
    },
    onSuccess: () => {
      toast.success('Campanha criada (rascunho)')
      setSheetOpen(false)
      setName('')
      setChannel('whatsapp')
      void qc.invalidateQueries({ queryKey: ['campaigns'] })
    },
    onError: (e: unknown) => {
      if (e instanceof ApiEnvelopeError) toast.error(e.message)
      else toast.error('Falha ao criar campanha')
    },
  })

  return (
    <div className="p-6 h-full flex flex-col gap-4 min-h-0 overflow-auto">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Campanhas</h1>
          <p className="text-sm text-text-muted">
            Lista e rascunhos na base de dados. Envio em massa e agendamento ficam para uma fase seguinte (fila + rate
            limit).
          </p>
        </div>
        <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
          <SheetTrigger asChild>
            <Button className="bg-primary">
              <Plus className="size-4" />
              Nova campanha
            </Button>
          </SheetTrigger>
          <SheetContent className="bg-card border-border w-full sm:max-w-md">
            <SheetHeader>
              <SheetTitle>Nova campanha (rascunho)</SheetTitle>
            </SheetHeader>
            <div className="mt-6 space-y-4">
              <div className="space-y-2">
                <Label htmlFor="camp-name">Nome</Label>
                <Input id="camp-name" value={name} onChange={(e) => setName(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="camp-ch">Canal</Label>
                <Input
                  id="camp-ch"
                  value={channel}
                  onChange={(e) => setChannel(e.target.value)}
                  placeholder="whatsapp"
                />
              </div>
              <Button
                className="w-full"
                disabled={createMut.isPending || !name.trim()}
                onClick={() => createMut.mutate()}
              >
                {createMut.isPending ? 'A criar…' : 'Criar rascunho'}
              </Button>
            </div>
          </SheetContent>
        </Sheet>
      </div>

      <div
        role="alert"
        className="rounded-lg border border-warning/40 bg-warning/10 px-4 py-3 text-sm text-text-primary"
      >
        <strong className="text-warning">LGPD e WhatsApp:</strong> confirme opt-in dos contatos e cumpra as políticas
        antes de qualquer disparo. Esta interface não substitui assessoria jurídica.
      </div>

      <Tabs defaultValue="campanhas" className="flex-1 flex flex-col min-h-0">
        <TabsList className="bg-card border border-border w-fit">
          <TabsTrigger value="campanhas">Campanhas</TabsTrigger>
          <TabsTrigger value="rapidos">Disparos rápidos</TabsTrigger>
          <TabsTrigger value="agendados">Agendados</TabsTrigger>
        </TabsList>
        <TabsContent value="campanhas" className="mt-4 flex-1">
          <div className="rounded-xl border border-border bg-card overflow-hidden">
            {isLoading ? (
              <div className="p-4 space-y-2">
                <Skeleton className="h-10 w-full" />
              </div>
            ) : data.length === 0 ? (
              <p className="p-6 text-sm text-text-muted">Nenhuma campanha. Cria um rascunho com &quot;Nova campanha&quot;.</p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow className="border-border hover:bg-transparent">
                    <TableHead>Nome</TableHead>
                    <TableHead>Canal</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Enviadas</TableHead>
                    <TableHead>Entregues</TableHead>
                    <TableHead>Lidas</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.map((c) => (
                    <TableRow key={c.id} className="border-border">
                      <TableCell className="font-medium">{c.name}</TableCell>
                      <TableCell>{c.channel}</TableCell>
                      <TableCell>
                        <Badge variant="secondary">{c.status}</Badge>
                      </TableCell>
                      <TableCell>{c.sent}</TableCell>
                      <TableCell>{c.delivered}</TableCell>
                      <TableCell>{c.read}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </div>
        </TabsContent>
        <TabsContent value="rapidos" className="mt-4 text-text-muted text-sm">
          Disparos rápidos não estão implementados. Para contactar um número, usa a{' '}
          <span className="text-text-primary font-medium">Inbox</span> (Evolution).
        </TabsContent>
        <TabsContent value="agendados" className="mt-4 text-text-muted text-sm">
          Agendamento de campanhas não está implementado — fase seguinte com fila e limites de envio.
        </TabsContent>
      </Tabs>
    </div>
  )
}
