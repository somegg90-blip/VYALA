import { createClient } from '@supabase/supabase-js'

const supabaseUrl = import.meta.env.VITE_SUPABASE_URL as string
const supabaseAnonKey = import.meta.env.VITE_SUPABASE_ANON_KEY as string

export const supabase = createClient(supabaseUrl, supabaseAnonKey)

export interface WishlistRow {
  id: string
  created_at: string
  name: string | null
  email: string | null
  role: string | null
  stack: string | null
  system_need: string | null
  wish: string | null
}