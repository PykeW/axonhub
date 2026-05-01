import { createContext } from 'react';
import type { Language, TranslationKey } from './i18n';

export interface LanguageContextType {
  language: Language;
  setLanguage: (language: Language) => void;
  t: (key: TranslationKey) => string;
}

export const LanguageContext = createContext<LanguageContextType | undefined>(undefined);
