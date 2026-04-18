import { useState } from 'react'
import {
  DragDropContext,
  Droppable,
  Draggable,
  type DropResult,
} from '@hello-pangea/dnd'
import { MessageSquare } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'

type Card = {
  id: string
  title: string
  phone: string
  value?: number
  tags: string[]
}

const initial: Record<string, Card[]> = {
  novo: [],
  qualificado: [],
  proposta: [],
  fechado: [],
}

const columns: { id: string; title: string }[] = [
  { id: 'novo', title: 'Novo lead' },
  { id: 'qualificado', title: 'Qualificado' },
  { id: 'proposta', title: 'Proposta' },
  { id: 'fechado', title: 'Fechado' },
]

export default function Kanban() {
  const [cols, setCols] = useState(initial)

  function onDragEnd(result: DropResult) {
    const { source, destination } = result
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
  }

  return (
    <div className="p-6 h-full flex flex-col gap-4 min-h-0">
      <div className="flex items-center justify-between shrink-0">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Pipeline CRM</h1>
          <p className="text-sm text-text-muted">Arraste cards entre etapas (mock)</p>
        </div>
        <Button variant="outline" size="sm">
          + Coluna
        </Button>
      </div>

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
                            {card.value != null && (
                              <div className="text-xs text-success mt-2">
                                R$ {card.value.toLocaleString('pt-BR')}
                              </div>
                            )}
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
                            >
                              <MessageSquare className="size-4" />
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
