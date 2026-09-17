/**
 * An error whose message is written for the person using the app: shown as is
 * instead of a generic "something went wrong". Throw it only with wording the
 * user can act on — never with an internal detail.
 */
export class UserFacingError extends Error {
  constructor(message: string, options?: { cause?: unknown }) {
    super(message, options);
    this.name = "UserFacingError";
  }
}
