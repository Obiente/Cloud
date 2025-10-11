export interface LabelValueItem {
  label: string;
  value: string | number;
}

export interface NameItem {
  name: string;
}

export type SelectItem = LabelValueItem | NameItem;
export type Booleanish = boolean | 'true' | 'false';
