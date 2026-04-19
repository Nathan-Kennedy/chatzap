import { useEffect, useState } from 'react'
import {
  DragDropContext,
  Droppable,
  Draggable,
  type DropResult,
} from '@hello-pangea/dnd'
import { MessageSquare } from 'lucide-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { api, unwrapEnvelope } from '@/lib/api'
import { ApiEnvelopeError } from '@/types/api'
import { toast } from 'sonner'

type Card = {
  id: string
  title: string
  phone: string
  tags: string[]
}

const columns: { id: string; title: string }[] = [
  { id: 'novo', title: 'Novo lead' },
  { id: 'qualificado', title: 'Qualificado' },
  { id: 'proposta', title: 'Proposta' },
  { id: 'fechado', title: 'Fechado' },
]

type KanbanBoardPayload = {
  stages: Record<string, Card[]>
}

function emptyCols(): Record<string, Card[]> {
  const o: Record<string, Card[]> = {}
  for (const c of columns) {
    o[c.id] = []
  }
  return o
}

async function fetchBoard(): Promise<Record<string, Card[]>> {
  const res = await api.get<unknown>('/kanban/board')
  const { data } = unwrapEnvelope<KanbanBoardPayload>(res)
  const next = emptyCols()
  for (const c of columns) {
    const list = data.stages?.[c.id]
    if (Array.isArray(list)) {
      next[c.id] = list.map((x) => ({
        id: x.id,
        title: x.title || '—',
        phone: x.phone || '',
        tags: Array.isArray(x.tags) ? x.tags : [],
      }))
    }
  }
  return next
}

export default function Kanban() {
  const qc = useQueryClient()
  const [cols, setCols] = useState<Record<string, Card[]>>(emptyCols)

  const { data: serverCols, isLoading } = useQuery({
    queryKey: ['kanban', 'board'],
    queryFn: fetchBoard,
  })

  useEffect(() => {
    if (serverCols) {
      setCols(serverCols)
    }
  }, [serverCols])

  const moveMut = useMutation({
    mutationFn: async ({ conversationId, stage }: { conversationId: string; stage: string }) => {
      const res = await api.patch<unknown>(`/kanban/cards/${conversationId}`, { stage })
      return unwrapEnvelope<{ id: string; pipeline_stage: string }>(res).data
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['kanban', 'board'] })
    },
    onError: (e: unknown) => {
      if (e instanceof ApiEnvelopeError) toast.error(e.message)
      else toast.error('Falha ao mover card')
      void qc.invalidateQueries({ queryKey: ['kanban', 'board'] })
    },
  })

  function onDragEnd(result: DropResult) {
    const { source, destination, draggableId } = result
    if (!destination) return
    if (
      source.droppableId === destination.droppableId &&
      source.index === destination.index
    ) {
      return
    }

    setCols((prev) => {
      const next = { ...prev }
      const from = [...(next[source.droppableId] ?? [])]
      const [moved] = from.splice(source.index, 1)
      if (!moved) return prev
      next[source.droppableId] = from
      const to = [...(next[destination.droppableId] ?? [])]
      to.splice(destination.index, 0, moved)
      next[destination.droppableId] = to
      return next
    })

    moveMut.mutate({
      conversationId: draggableId,
      stage: destination.droppableId,
    })
  }

  return (
    <div className="p-6 h-full flex flex-col gap-4 min-h-0">
      <div className="flex items-center justify-between shrink-0">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Pipeline CRM</h1>
          <p className="text-sm text-text-muted">
            Conversas do workspace por etapa (guardado no servidor). Arraste para alterar.
          </p>
        </div>
        <Button variant="outline" size="sm" asChild>
          <Link to="/inbox">Abrir Inbox</Link>
        </Button>
      </div>

      {isLoading ? (
        <p className="text-sm text-text-muted">A carregar…</p>
      ) : null}

      <DragDropContext onDragEnd={onDragEnd}>
        <div className="flex gap-4 overflow-x-auto pb-2 flex-1 min-h-0">
          {columns.map((col) => (
            <Droppable key={col.id} droppableId={col.id}>
              {(provided, snapshot) => (
                <div
                  ref={provided.innerRef}
                  {...provided.droppableProps}
                  className={cn(
                    'w-[280px] shrink-0 rounded-xl border border-border bg-sidebar/50 flex flex-col min-h-[320px]',
                    snapshot.isDraggingOver && 'ring-1 ring-primary/40'
                  )}
                >
                  <div className="p-3 border-b border-border flex items-center justify-between">
                    <span className="font-medium text-sm">{col.title}</span>
                    <Badge variant="secondary">{cols[col.id]?.length ?? 0}</Badge>
                  </div>
                  <div className="p-2 flex flex-col gap-2 flex-1">
                    {(cols[col.id] ?? []).map((card, index) => (
                      <Draggable key={card.id} draggableId={card.id} index={index}>
                        {(dragProvided, dragSnapshot) => (
                          <div
                            ref={dragProvided.innerRef}
                            {...dragProvided.draggableProps}
                            {...dragProvided.dragHandleProps}
                            className={cn(
                              'rounded-lg border border-border bg-card p-3 text-sm shadow-sm cursor-grab active:cursor-grabbing',
                              dragSnapshot.isDragging && 'shadow-lg ring-1 ring-primary/30'
                            )}
                          >
                            <div className="font-medium">{card.title}</div>
                            <div className="text-text-muted text-xs mt-1">
                              {card.phone}
                            </div>
                            <div className="flex gap-1 mt-2 flex-wrap">
                              {card.tags.map((t) => (
                                <Badge key={t} variant="outline" className="text-[10px]">
                                  {t}
                                </Badge>
                              ))}
                            </div>
                            <Button
                              type="button"
                              variant="ghost"
                              size="icon"
                              className="h-7 w-7 mt-2 text-green-500"
                              asChild
                            >
                              <Link to="/inbox" title="Abrir caixa de entrada">
                                <MessageSquare className="size-4" />
                              </Link>
                            </Button>
                          </div>
                        )}
                      </Draggable>
                    ))}
                    {provided.placeholder}
                  </div>
                </div>
              )}
            </Droppable>
          ))}
        </div>
      </DragDropContext>
    </div>
  )
}
