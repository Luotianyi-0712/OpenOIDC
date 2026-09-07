export type AnnouncementMode = 'banner' | 'modal' | 'both'

export interface AnnouncementInput {
  title: string
  content: string
  display_mode: AnnouncementMode
  dismissible: boolean
  scrolling: boolean
  is_published: boolean
}

export interface Announcement extends AnnouncementInput {
  id: string
  revision: string
  created_at: string
  updated_at: string
}
