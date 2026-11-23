/*
Copyright 2026 The Tekton Authors & Platform Team
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import { useIntl } from 'react-intl';
import { HeaderGlobalAction } from '@carbon/react';
import { Translate as TranslateIcon } from '@carbon/react/icons';

import { getLocale, setLocale } from '../../utils';

export default function LanguageSwitcher() {
  const intl = useIntl();
  const currentLocale = getLocale();
  const isZh = currentLocale.startsWith('zh');

  const toggleLanguage = () => {
    const nextLocale = isZh ? 'en' : 'zh-Hans';
    setLocale(nextLocale);
  };

  const tooltipText = isZh
    ? intl.formatMessage({
        id: 'dashboard.header.switchToEnglish',
        defaultMessage: '切换至英文 (Switch to English)'
      })
    : intl.formatMessage({
        id: 'dashboard.header.switchToChinese',
        defaultMessage: '切换至中文 (Switch to Chinese)'
      });

  return (
    <HeaderGlobalAction
      aria-label={tooltipText}
      className="tkn--lang-switcher-btn"
      onClick={toggleLanguage}
      tooltipAlignment="end"
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          gap: '5px',
          height: '100%',
          padding: '0 6px',
          fontSize: '12px',
          fontWeight: 600,
          color: 'inherit'
        }}
      >
        <TranslateIcon size={18} />
        <span>{isZh ? '中' : 'EN'}</span>
      </div>
    </HeaderGlobalAction>
  );
}
