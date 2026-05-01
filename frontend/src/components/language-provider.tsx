import { useMemo, useState } from 'react';
import { translations, type Language, type TranslationKey } from './i18n';
import { LanguageContext } from './language-context';

export function LanguageProvider({ children }: { children: React.ReactNode }) {
  const [language, setLanguage] = useState<Language>('zh');

  const t = useMemo(
    () => (key: TranslationKey) => {
      return translations[language][key] || key;
    },
    [language]
  );

  const value = {
    language,
    setLanguage,
    t,
  };

  return <LanguageContext.Provider value={value}>{children}</LanguageContext.Provider>;
}
