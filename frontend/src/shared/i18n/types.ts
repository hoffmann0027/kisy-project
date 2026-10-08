// The shape of a dictionary. Russian is the source: every other language
// must carry every Russian key (Translation<> makes a missing one a compile
// error), so no screen can be half-translated.

/** A countable phrase, picked by the language's plural rules (Intl.PluralRules). */
export interface PluralMsg {
  zero?: string;
  one?: string;
  two?: string;
  few?: string;
  many?: string;
  other: string;
}

export type Msg = string | PluralMsg;

export type Dict = Record<string, Msg>;

/** The same keys as the Russian namespace `T`, in another language. */
export type Translation<T extends Dict> = { [K in keyof T]: Msg };
