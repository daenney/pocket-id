export type PaginationRequest = {
	page: number;
	limit: number;
};

export type SortRequest = {
	column: string;
	direction: 'asc' | 'desc';
};

export type FilterMap = Record<string, string>;

export type SearchPaginationSortRequest = {
	search?: string;
	pagination?: PaginationRequest;
	sort?: SortRequest;
	filters?: FilterMap;
};

export type PaginationResponse = {
	totalPages: number;
	totalItems: number;
	currentPage: number;
	itemsPerPage: number;
};

export type Paginated<T> = {
	data: T[];
	pagination: PaginationResponse;
};
