import { useQuery } from '@tanstack/react-query'
import { api, unwrapEnvelope } from '@/lib/api'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  BarChart,
  Bar,
} from 'recharts'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

type Overview = {
  messages_last_30d: number
  conversations_total: number
  instances_total: number
}

type TimeseriesDay = {
  date: string
  inbound: number
  outbound: number
}

type TimeseriesHour = {
  hour: number
  messages: number
}

type TimeseriesPayload = {
  by_day: TimeseriesDay[]
  by_hour: TimeseriesHour[]
  note?: string
}

async function fetchOverview(): Promise<Overview> {
  const res = await api.get<unknown>('/analytics/overview')
  const { data } = unwrapEnvelope<Overview>(res)
  return data
}

async function fetchTimeseries(): Promise<TimeseriesPayload> {
  const res = await api.get<unknown>('/analytics/timeseries')
  const { data } = unwrapEnvelope<TimeseriesPayload>(res)
  return data
}

function shortDayLabel(isoDate: string): string {
  const d = new Date(`${isoDate}T12:00:00Z`)
  if (Number.isNaN(d.getTime())) return isoDate
  return d.toLocaleDateString('pt-BR', { weekday: 'short', day: '2-digit', month: '2-digit' })
}

export default function Analytics() {
  const { data: overview } = useQuery({
    queryKey: ['analytics', 'overview'],
    queryFn: fetchOverview,
  })

  const { data: ts } = useQuery({
    queryKey: ['analytics', 'timeseries'],
    queryFn: fetchTimeseries,
  })

  const lineData =
    ts?.by_day?.map((r) => ({
      day: shortDayLabel(r.date),
      recebidas: Number(r.inbound),
      enviadas: Number(r.outbound),
    })) ?? []

  const barData =
    ts?.by_hour?.map((r) => ({
      h: `${r.hour}h`,
      vol: Number(r.messages),
    })) ?? []

  return (
    <div className="p-6 h-full overflow-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-text-primary">Analytics</h1>
        <p className="text-sm text-text-muted">Resumo do workspace (últimos 30 dias — mensagens)</p>
      </div>

      <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {[
          { t: 'Mensagens (30d)', v: String(overview?.messages_last_30d ?? '—') },
          { t: 'Conversas', v: String(overview?.conversations_total ?? '—') },
          { t: 'Instâncias', v: String(overview?.instances_total ?? '—') },
        ].map((k) => (
          <Card key={k.t} className="bg-card border-border">
            <CardHeader className="pb-1">
              <CardTitle className="text-sm font-medium text-text-muted">
                {k.t}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-2xl font-bold text-text-primary">{k.v}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="grid lg:grid-cols-2 gap-6">
        <Card className="bg-card border-border">
          <CardHeader>
            <CardTitle className="text-base">Mensagens por dia</CardTitle>
            <CardDescription className="text-xs">
              Recebidas (inbound) vs enviadas (outbound), UTC — dias com actividade apenas.
            </CardDescription>
          </CardHeader>
          <CardContent className="h-[260px]">
            {lineData.length === 0 ? (
              <p className="text-sm text-text-muted py-8 text-center">Sem dados no período.</p>
            ) : (
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={lineData}>
                  <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                  <XAxis dataKey="day" className="text-xs" />
                  <YAxis className="text-xs" />
                  <Tooltip
                    contentStyle={{
                      background: '#15151F',
                      border: '1px solid rgba(255,255,255,0.08)',
                    }}
                  />
                  <Line type="monotone" dataKey="recebidas" stroke="#7C3AED" name="Recebidas" />
                  <Line type="monotone" dataKey="enviadas" stroke="#10B981" name="Enviadas" />
                </LineChart>
              </ResponsiveContainer>
            )}
          </CardContent>
        </Card>

        <Card className="bg-card border-border">
          <CardHeader>
            <CardTitle className="text-base">Volume por hora do dia</CardTitle>
            <CardDescription className="text-xs">
              Total de mensagens por hora (0–23 UTC), agregado nos últimos 30 dias.
            </CardDescription>
          </CardHeader>
          <CardContent className="h-[260px]">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={barData}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                <XAxis dataKey="h" interval={2} className="text-xs" />
                <YAxis className="text-xs" />
                <Tooltip
                  contentStyle={{
                    background: '#15151F',
                    border: '1px solid rgba(255,255,255,0.08)',
                  }}
                />
                <Bar dataKey="vol" fill="#06B6D4" radius={[4, 4, 0, 0]} name="Mensagens" />
              </BarChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
