export interface MentionMatch {
  mention: string;
  username: string;
  index: number;
}

const MENTION_REGEX = /@(\w[\w.-]*)/g;

export function parseMentions(text: string): MentionMatch[] {
  const matches: MentionMatch[] = [];
  const regex = new RegExp(MENTION_REGEX.source, MENTION_REGEX.flags);
  let match = regex.exec(text);
  while (match !== null) {
    matches.push({
      mention: match[0],
      username: match[1],
      index: match.index,
    });
    match = regex.exec(text);
  }
  return matches;
}

export function highlightMentions(text: string): string {
  return text.replace(MENTION_REGEX, '<span class="bc-mention">@$1</span>');
}

export function extractMentionUsers(text: string): string[] {
  const matches = parseMentions(text);
  return [...new Set(matches.map(m => m.username))];
}
