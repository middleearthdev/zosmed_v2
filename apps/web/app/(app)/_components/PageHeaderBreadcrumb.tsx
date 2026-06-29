/**
 * Breadcrumb strip for the 56px PageHeader bar.
 * Used by: Analytics overview, Contact Profile, Analytics drilldown (3 screens → shared).
 *
 * Rendering:
 *   crumbs[0..n-2]  — muted (text-text-2), optional href
 *   crumbs[n-1]     — active (text-text, font-medium)
 *   separator       — "/" in text-text-3
 */
import Link from 'next/link';

export interface BreadcrumbCrumb {
  label: string;
  href?: string;
}

export function PageHeaderBreadcrumb({ crumbs }: { crumbs: BreadcrumbCrumb[] }) {
  return (
    <div className="flex items-center gap-2.5 text-[13px]">
      {crumbs.map((crumb, i) => {
        const isLast = i === crumbs.length - 1;
        return (
          <span key={`${crumb.label}-${i}`} className="flex items-center gap-2.5">
            {isLast ? (
              <span className="text-text font-medium">{crumb.label}</span>
            ) : crumb.href ? (
              <Link href={crumb.href} className="text-text-2 hover:text-text transition-colors">
                {crumb.label}
              </Link>
            ) : (
              <span className="text-text-2">{crumb.label}</span>
            )}
            {!isLast && <span className="text-text-3">/</span>}
          </span>
        );
      })}
    </div>
  );
}
