// The Russian dictionary: the source every other language translates.
import { common } from "./common";
import { admin } from "./admin";
import { chat } from "./chat";
import { account } from "./account";
import { work } from "./work";
import { hub } from "./hub";

export const ru = { ...common, ...admin, ...chat, ...account, ...work, ...hub };

/** Every key the app may ask for. */
export type Key = keyof typeof ru;
