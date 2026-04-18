import { Plus } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
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
  const { data = [], isLoading } = useQuery({
    queryKey: ['campaigns'],
    queryFn: fetchCampaigns,
  })

  return (
    <div className="p-6 h-full flex flex-col gap-4 min-h-0 overflow-auto">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Campanhas</h1>
          <p className="text-sm text-text-muted">Disparos e agendamentos</p>
        </div>
        <Button className="bg-primary">
          <Plus className="size-4" />
          Nova campanha
        </Button>
      </div>

      <div
        role="alert"
        className="rounded-lg border border-warning/40 bg-warning/10 px-4 py-3 text-sm text-text-primary"
      >
        <strong className="text-warning">LGPD e WhatsApp:</strong> confirme opt-in dos
        contatos e cumpra as políticas da Meta antes de enviar campanhas. Esta interface
        não substitui assessoria jurídica.
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
          Nenhum disparo rápido (mock).
        </TabsContent>
        <TabsContent value="agendados" className="mt-4 text-text-muted text-sm">
          Nenhum agendamento (mock).
        </TabsContent>
      </Tabs>
    </div>
  )
}
