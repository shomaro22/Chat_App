export type RoomSummary = {
  id: string;
  name?: string | null;
  direct_peer_name?: string | null;
  unread_count?: number;
  last_message?: {
    preview?: string | null;
  };
};

export type Message = {
  id: string;
  room_id: string;
  parent_message_id?: string | null;
  content?: string | null;
  image_url?: string | null;
  file_url?: string | null;
  file_name?: string | null;
  file_content_type?: string | null;
  sender_id?: string;
  sender_name?: string;
  created_at: string;
  updated_at?: string;
  is_deleted?: boolean;
  deleted_at?: string | null;
  /** GET .../messages?include_read_count=1 のとき（グループの既読人数） */
  read_count?: number | null;
};

export type ReactionRow = {
  user_id: string;
  emoji: string;
  created_at: string;
};

export type ReactionChip = {
  emoji: string;
  count: number;
  mine: boolean;
  userNames?: string[];
};

export type RoomMemberRow = {
  user_id: string;
  name: string;
  online: boolean;
};

export type ReadStateResponse = {
  my_last_read_message_id?: string;
  unread_count?: number;
  member_reads?: Array<{
    user_id: string;
    user_name?: string;
    last_read_message_id?: string;
  }>;
};

export type ReaderInfo = {
  userId: string;
  userName?: string;
  lastReadMessageId: string;
};

export type MeUser = {
  id: string;
  email: string;
  name: string;
  role: string;
  active?: boolean;
  email_verified?: boolean;
};

export const MESSAGE_PAGE_LIMIT = 50;
