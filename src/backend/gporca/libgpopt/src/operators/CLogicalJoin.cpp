//---------------------------------------------------------------------------
//	Greenplum Database
//	Copyright (C) 2018 VMware, Inc. or its affiliates.
//
//	@filename:
//		CLogicalJoin.cpp
//
//	@doc:
//		Implementation of logical join class
//---------------------------------------------------------------------------

#include "gpopt/operators/CLogicalJoin.h"

#include "gpos/base.h"

#include "gpopt/base/CColRefSet.h"
#include "gpopt/base/COptCtxt.h"
#include "gpopt/operators/CExpression.h"
#include "gpopt/operators/CExpressionHandle.h"
#include "gpopt/operators/CPredicateUtils.h"
#include "naucrates/statistics/CJoinStatsProcessor.h"

using namespace gpopt;


//---------------------------------------------------------------------------
//	@function:
//		CLogicalJoin::CLogicalJoin
//
//	@doc:
//		Ctor
//
//---------------------------------------------------------------------------

CLogicalJoin::CLogicalJoin(CMemoryPool *mp, CXform::EXformId origin_xform)
	: CLogical(mp), m_origin_xform(origin_xform)
{
	GPOS_ASSERT(nullptr != mp);
}

//---------------------------------------------------------------------------
//	@function:
//		CLogicalJoin::Matches
//
//	@doc:
//		Match function on operator level
//
//---------------------------------------------------------------------------
BOOL
CLogicalJoin::Matches(COperator *pop) const
{
	return (pop->Eopid() == Eopid());
}

CTableDescriptorHashSet *
CLogicalJoin::DeriveTableDescriptor(CMemoryPool *mp,
									CExpressionHandle &exprhdl) const
{
	CTableDescriptorHashSet *table_descriptor_set =
		GPOS_NEW(mp) CTableDescriptorHashSet(mp);

	for (ULONG ul = 0; ul < exprhdl.Arity(); ul++)
	{
		if (exprhdl.Pop(ul)->FLogical())
		{
			CTableDescriptorHashSetIter hsiter(
				exprhdl.DeriveTableDescriptor(ul));
			while (hsiter.Advance())
			{
				CTableDescriptor *ptabdesc =
					const_cast<CTableDescriptor *>(hsiter.Get());
				if (table_descriptor_set->Insert(ptabdesc))
				{
					ptabdesc->AddRef();
				}
			}
		}
	}
	return table_descriptor_set;
}

//---------------------------------------------------------------------------
//	@function:
//		CLogicalJoin::PstatsDerive
//
//	@doc:
//		Derive statistics
//
//---------------------------------------------------------------------------
IStatistics *
CLogicalJoin::PstatsDerive(CMemoryPool *mp, CExpressionHandle &exprhdl,
						   IStatisticsArray *stats_ctxt) const
{
	IStatistics *pstats =
		CJoinStatsProcessor::DeriveJoinStats(mp, exprhdl, stats_ctxt);

	// Check whether a row plan hint exists for this join operators relations.
	// And if one does exist, then evaluate the hint to overwrite the estimated
	// rows.
	CPlanHint *planhint =
		COptCtxt::PoctxtFromTLS()->GetOptimizerConfig()->GetPlanHint();
	if (nullptr != planhint)
	{
		CRowHint *rowhint =
			planhint->GetRowHint(exprhdl.DeriveTableDescriptor());
		if (nullptr != rowhint)
		{
			pstats->SetRows(rowhint->ComputeRows(pstats->Rows()));
		}
	}
	return pstats;
}

// EOF
