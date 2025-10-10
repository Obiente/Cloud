import { eventHandler } from 'h3';
import { getUserData } from '../../utils/auth';

export default eventHandler(async event => {
  const session = await getUserSession(event);
  // Populate user data if session exists
  if (Object.keys(session).length > 0) {
    await getUserData(event, session);
  }
  // Exclude secure (server-only) data from response
  const { secure, ...data } = session;
  return data;
});
