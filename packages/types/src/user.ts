export interface User {
  /** User ID */
  sub: string;
  name: string;
  given_name: string;
  family_name: string;
  locale: string;
  updated_at: number;
  preferred_username: string;
  email: string;
  email_verified: boolean;
}

export interface Organization {
  id: string;
  name: string;
  slug: string;
  createdAt: Date;
  updatedAt: Date;
}
