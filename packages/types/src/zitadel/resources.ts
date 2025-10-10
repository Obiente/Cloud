/**
 * Zitadel user profile
 * @see https://zitadel.com/docs/apis/resources/user
 */
export interface ZitadelUserProfile {
  user_id: string;
  details: {
    resource_owner: string;
    created_at: string;
    sequence: number;
    changed_at: string;
    state: string;
  };
  human: {
    profile: {
      first_name: string;
      last_name: string;
      display_name: string;
      preferred_language?: string;
      gender?: string;
      email: string;
      phone?: string;
      avatar_url?: string;
    };
    email: {
      email: string;
      is_email_verified: boolean;
    };
  };
}

/**
 * Zitadel organization
 * @see https://zitadel.com/docs/apis/resources/org
 */
export interface ZitadelOrganization {
  id: string;
  details: {
    sequence: number;
    creation_date: string;
    change_date: string;
    resource_owner: string;
  };
  state: string;
  name: string;
  primary_domain?: string;
  domains?: string[];
}

/**
 * Zitadel project
 * @see https://zitadel.com/docs/apis/resources/project
 */
export interface ZitadelProject {
  id: string;
  details: {
    sequence: number;
    creation_date: string;
    change_date: string;
    resource_owner: string;
  };
  state: string;
  name: string;
  project_role_assertion: boolean;
  project_role_check: boolean;
  has_project_check: boolean;
  private_labeling_setting: string;
}

/**
 * Zitadel role
 * @see https://zitadel.com/docs/apis/resources/project#roles
 */
export interface ZitadelRole {
  role_key: string;
  role_display_name: string;
  role_group?: string;
  description?: string;
  details: {
    sequence: number;
    creation_date: string;
    change_date: string;
    resource_owner: string;
  };
}
