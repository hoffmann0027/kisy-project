import { common } from "./common";
import { admin } from "./admin";
import { chat } from "./chat";
import { account } from "./account";
import { work } from "./work";
import { hub } from "./hub";

export const messages = { ...common, ...admin, ...chat, ...account, ...work, ...hub };
