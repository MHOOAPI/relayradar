export interface GuideSummary {
  slug: string;
  title: string;
  desc: string;
  updated: string;
  minutes: number;
}

export interface GuideDetail extends GuideSummary {
  bodyHtml: string;
}

export interface GuideEN extends GuideSummary {
  bodyHtml?: string;
}
