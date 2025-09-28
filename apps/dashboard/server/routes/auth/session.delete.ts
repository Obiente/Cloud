import { eventHandler } from 'h3';
import { clearUserSession } from '~~/server/utils/session';

export default eventHandler(async event => {
  return await clearUserSession(event);
});
