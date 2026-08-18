// i18n 翻译 key 类型定义:所有页面/组件文本集中管理。

export type Locale = 'zh-CN' | 'en-US' | 'ms-MY'

export interface Translations {
  common: {
    brand: string
    tagline: string
    footer: string
    logout: string
    profile: string
    loading: string
    dataScope: string
    dataScopeAll: string
  }
  auth: {
    login: {
      title: string
      subtitle: string
      usernamePlaceholder: string
      passwordPlaceholder: string
      submit: string
      submitting: string
      noAccount: string
      toRegister: string
      emptyFields: string
      fail: string
    }
    register: {
      title: string
      subtitle: string
      namePlaceholder: string
      usernamePlaceholder: string
      passwordPlaceholder: string
      confirmPlaceholder: string
      submit: string
      submitting: string
      hasAccount: string
      toLogin: string
      emptyName: string
      invalidUsername: string
      shortPassword: string
      mismatch: string
      fail: string
      tip: string
    }
  }
  ads: Array<{ title: string; sub: string }>
  shell: {
    home: string
    searchPlaceholder: string
    notifications: string
    themeToDark: string
    themeToLight: string
    collapseMenu: string
    expandMenu: string
    quickCreate: string
    version: string
  }
  menu: {
    groups: Record<string, string>
    items: Record<string, string>
  }
  pages: {
    dashboard: {
      title: string
      welcome: string
    }
    placeholder: {
      building: string
    }
    geo: {
      title: string
      tabCountry: string
      tabSubdiv: string
      countryColumns: string[]
      subdivColumns: string[]
      add: string
      edit: string
      save: string
      cancel: string
      disable: string
      enable: string
      detail: string
      active: string
      inactive: string
      total: string
      loadFail: string
      saveFail: string
      filterCountry: string
      searchPlaceholder: string
      empty: string
      prev: string
      next: string
      perPage: string
      rangeText: string
      jumpText: string
      pageUnit: string
      names: string
      addName: string
      attrs: string
      timeZones: string
      currencies: string
      callingCodes: string
      countryFields: {
        alpha2: string
        alpha3: string
        numericCode: string
        shortName: string
        fullName: string
        m49Region: string
        postalRegex: string
        continent: string
        status: string
      }
      subdivFields: {
        code: string
        countryCode: string
        parentCode: string
        category: string
        level: string
        osmAdminLevel: string
        geonameId: string
      }
    }
    address: {
      title: string
      expand: string
      collapse: string
      attach: string
      unlinked: string
      unlinkedAll: string
      searchAll: string
      noHit: string
      addRoot: string
      addChild: string
      rename: string
      delete: string
      deleteConfirm: string
      deleteFail: string
      pathLabel: string
      pathHint: string
      nameLabel: string
      country: string
      adminCode: string
      none: string
      loadFail: string
      saveFail: string
      empty: string
    }
    importer: {
      title: string
      addrTitle: string
      addrHint: string
      geoTitle: string
      geoHint: string
      importBtn: string
      parseFail: string
      imported: string
      loadFail: string
    }
    account: {
      title: string
      searchPlaceholder: string
      columns: string[]
      edit: string
      allGroup: string
      total: string
      loadFail: string
    }
    error: {
      forbidden: string
      notFound: string
    }
  }
}