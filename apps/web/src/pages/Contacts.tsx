import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Link } from 'react-router-dom'
import { MessageSquare, Plus, Search } from 'lucide-react'
import { useContacts } from '@/hooks/useContacts'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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
import { formatRelativeShort } from '@/utils/format'
import { toast } from 'sonner'

const contactSchema = z.object({
  name: z.string().min(2, 'Nome obrigatório'),
  phone: z.string().min(8, 'Telefone inválido'),
  email: z.string().email('E-mail inválido').optional().or(z.literal('')),
})

type ContactForm = z.infer<typeof contactSchema>

const SEARCH_DEBOUNCE_MS = 300

export default function Contacts() {
  const [searchInput, setSearchInput] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')

  useEffect(() => {
    const t = window.setTimeout(() => setDebouncedSearch(searchInput.trim()), SEARCH_DEBOUNCE_MS)
    return () => window.clearTimeout(t)
  }, [searchInput])

  const { data: rows = [], isLoading } = useContacts(debouncedSearch || undefined)
  const form = useForm<ContactForm>({
    resolver: zodResolver(contactSchema),
    defaultValues: { name: '', phone: '', email: '' },
  })

  function onSubmit(values: ContactForm) {
    toast.info(
      `Contacto "${values.name}" — CRM em evolução. Para WhatsApp, usa a Caixa de entrada e "Nova conversa".`
    )
    form.reset()
  }

  return (
    <div className="p-6 h-full flex flex-col gap-4 min-h-0 overflow-auto">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Contatos</h1>
          <p className="text-sm text-text-muted max-w-xl">
            Lista sincronizada com a API de contactos. Para abrir conversa WhatsApp com um número, usa a{' '}
            <Link to="/inbox" className="text-primary underline font-medium">
              Caixa de entrada
            </Link>{' '}
            (Nova conversa).
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" className="border-border" asChild>
            <Link to="/inbox" className="gap-2">
              <MessageSquare className="size-4" />
              Conversas WhatsApp
            </Link>
          </Button>
          <div className="relative flex-1 min-w-[200px]">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted size-4" />
            <Input
              placeholder="Buscar por nome ou telefone…"
              className="pl-9 bg-card"
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              aria-label="Buscar contactos"
            />
          </div>
          <Sheet>
            <SheetTrigger asChild>
              <Button className="bg-primary">
                <Plus className="size-4" />
                Novo contato
              </Button>
            </SheetTrigger>
            <SheetContent className="bg-card border-border w-full sm:max-w-md">
              <SheetHeader>
                <SheetTitle>Novo contato</SheetTitle>
              </SheetHeader>
              <form
                className="mt-6 space-y-4"
                onSubmit={form.handleSubmit(onSubmit)}
              >
                <div className="space-y-2">
                  <Label htmlFor="c-name">Nome</Label>
                  <Input id="c-name" {...form.register('name')} />
                  {form.formState.errors.name && (
                    <p className="text-xs text-destructive">
                      {form.formState.errors.name.message}
                    </p>
                  )}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="c-phone">Telefone</Label>
                  <Input id="c-phone" {...form.register('phone')} />
                  {form.formState.errors.phone && (
                    <p className="text-xs text-destructive">
                      {form.formState.errors.phone.message}
                    </p>
                  )}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="c-email">E-mail</Label>
                  <Input id="c-email" type="email" {...form.register('email')} />
                  {form.formState.errors.email && (
                    <p className="text-xs text-destructive">
                      {form.formState.errors.email.message}
                    </p>
                  )}
                </div>
                <Button type="submit" className="w-full">
                  Guardar (CRM em breve)
                </Button>
                <Button type="button" variant="secondary" className="w-full" asChild>
                  <Link to="/inbox">Abrir Caixa de entrada</Link>
                </Button>
              </form>
            </SheetContent>
          </Sheet>
        </div>
      </div>

      <div className="rounded-xl border border-border bg-card overflow-hidden flex-1 min-h-[200px]">
        {isLoading ? (
          <div className="p-4 space-y-2">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow className="border-border hover:bg-transparent">
                <TableHead>Nome</TableHead>
                <TableHead>Telefone</TableHead>
                <TableHead>Canal</TableHead>
                <TableHead>Última interação</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((c) => (
                <TableRow key={c.id} className="border-border">
                  <TableCell className="font-medium">{c.name}</TableCell>
                  <TableCell className="text-text-muted">{c.phone}</TableCell>
                  <TableCell>
                    <Badge variant="secondary" className="capitalize">
                      {c.channel}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-text-muted text-sm">
                    {c.last_seen_at
                      ? formatRelativeShort(c.last_seen_at)
                      : '—'}
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
