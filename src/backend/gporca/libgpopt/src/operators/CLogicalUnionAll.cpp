//---------------------------------------------------------------------------
//	Greenplum Database
//	Copyright (C) 2012 EMC Corp.
//
//	@filename:
//		CLogicalUnionAll.cpp
//
//	@doc:
//		Implementation of UnionAll operator
//---------------------------------------------------------------------------

#include "gpopt/operators/CLogicalUnionAll.h"

#include "gpos/base.h"

#include "gpopt/base/CUtils.h"
#include "gpopt/operators/CExpressionHandle.h"
#include "naucrates/statistics/CUnionAllStatsProcessor.h"

using namespace gpopt;

//---------------------------------------------------------------------------
//	@function:
//		CLogicalUnionAll::CLogicalUnionAll
//
//	@doc:
//		Ctor - for pattern
//
//---------------------------------------------------------------------------
CLogicalUnionAll::CLogicalUnionAll(CMemoryPool *mp) : CLogicalUnion(mp)
{
	m_fPattern = true;
}

//---------------------------------------------------------------------------
//	@function:
//		CLogicalUnionAll::CLogicalUnionAll
//
//	@doc:
//		Ctor
//
//---------------------------------------------------------------------------
CLogicalUnionAll::CLogicalUnionAll(CMemoryPool *mp, CColRefArray *pdrgpcrOutput,
								   CColRef2dArray *pdrgpdrgpcrInput)
	: CLogicalUnion(mp, pdrgpcrOutput, pdrgpdrgpcrInput)
{
}

//---------------------------------------------------------------------------
//	@function:
//		CLogicalUnionAll::~CLogicalUnionAll
//
//	@doc:
//		Dtor
//
//---------------------------------------------------------------------------
CLogicalUnionAll::~CLogicalUnionAll() = default;

//---------------------------------------------------------------------------
//	@function:
//		CLogicalUnionAll::DeriveMaxCard
//
//	@doc:
//		Derive max card
//
//---------------------------------------------------------------------------
CMaxCard
CLogicalUnionAll::DeriveMaxCard(CMemoryPool *,	// mp
								CExpressionHandle &exprhdl) const
{
	const ULONG arity = exprhdl.Arity();

	CMaxCard maxcard = exprhdl.DeriveMaxCard(0);
	for (ULONG ul = 1; ul < arity; ul++)
	{
		maxcard += exprhdl.DeriveMaxCard(ul);
	}

	return maxcard;
}

//---------------------------------------------------------------------------
//	@function:
//		CLogicalUnionAll::PopCopyWithRemappedColumns
//
//	@doc:
//		Return a copy of the operator with remapped columns
//
//---------------------------------------------------------------------------
COperator *
CLogicalUnionAll::PopCopyWithRemappedColumns(CMemoryPool *mp,
											 UlongToColRefMap *colref_mapping,
											 BOOL must_exist)
{
	CColRefArray *pdrgpcrOutput =
		CUtils::PdrgpcrRemap(mp, m_pdrgpcrOutput, colref_mapping, must_exist);
	CColRef2dArray *pdrgpdrgpcrInput = CUtils::PdrgpdrgpcrRemap(
		mp, m_pdrgpdrgpcrInput, colref_mapping, must_exist);

	return GPOS_NEW(mp) CLogicalUnionAll(mp, pdrgpcrOutput, pdrgpdrgpcrInput);
}

//---------------------------------------------------------------------------
//	@function:
//		CLogicalUnionAll::PkcDeriveKeys
//
//	@doc:
//		Derive key collection
//
//---------------------------------------------------------------------------
CKeyCollection *
CLogicalUnionAll::DeriveKeyCollection(CMemoryPool *,	   //mp,
									  CExpressionHandle &  // exprhdl
) const
{
	return nullptr;
}

//---------------------------------------------------------------------------
//	@function:
//		CLogicalUnionAll::PxfsCandidates
//
//	@doc:
//		Get candidate xforms
//
//---------------------------------------------------------------------------
CXformSet *
CLogicalUnionAll::PxfsCandidates(CMemoryPool *mp) const
{
	CXformSet *xform_set = GPOS_NEW(mp) CXformSet(mp);
	(void) xform_set->ExchangeSet(CXform::ExfImplementUnionAll);

	return xform_set;
}


//---------------------------------------------------------------------------
//	@function:
//		CLogicalUnionAll::PstatsDeriveUnionAll
//
//	@doc:
//		Derive statistics based on union all semantics
//
//---------------------------------------------------------------------------
IStatistics *
CLogicalUnionAll::PstatsDeriveUnionAll(CMemoryPool *mp,
									   CExpressionHandle &exprhdl,
									   CColRefArray *pdrgpcrOutput,
									   CColRef2dArray *pdrgpdrgpcrInput)
{
	GPOS_ASSERT(COperator::EopLogicalUnionAll == exprhdl.Pop()->Eopid() ||
				COperator::EopLogicalUnion == exprhdl.Pop()->Eopid());

	GPOS_ASSERT(nullptr != pdrgpcrOutput);
	GPOS_ASSERT(nullptr != pdrgpdrgpcrInput);

	IStatistics *result_stats = exprhdl.Pstats(0);
	result_stats->AddRef();
	const ULONG arity = exprhdl.Arity();
	for (ULONG ul = 1; ul < arity; ul++)
	{
		IStatistics *child_stats = exprhdl.Pstats(ul);
		CStatistics *stats = CUnionAllStatsProcessor::CreateStatsForUnionAll(
			mp, dynamic_cast<CStatistics *>(result_stats),
			dynamic_cast<CStatistics *>(child_stats),
			CColRef::Pdrgpul(mp, pdrgpcrOutput),
			CColRef::Pdrgpul(mp, (*pdrgpdrgpcrInput)[0]),
			CColRef::Pdrgpul(mp, (*pdrgpdrgpcrInput)[ul]));
		result_stats->Release();
		result_stats = stats;
	}

	return result_stats;
}

//---------------------------------------------------------------------------
//	@function:
//		CLogicalUnionAll::PstatsDerive
//
//	@doc:
//		Derive statistics based on union all semantics
//
//---------------------------------------------------------------------------
IStatistics *
CLogicalUnionAll::PstatsDerive(CMemoryPool *mp, CExpressionHandle &exprhdl,
							   IStatisticsArray *  // not used
) const
{
	GPOS_ASSERT(EspNone < Esp(exprhdl));
	CColRefArray *pdrgpcrOutput =
		CLogicalSetOp::PopConvert(exprhdl.Pop())->PdrgpcrOutput();
	CColRef2dArray *pdrgpdrgpcrInput =
		CLogicalSetOp::PopConvert(exprhdl.Pop())->PdrgpdrgpcrInput();

	CReqdPropRelational *prprel =
		CReqdPropRelational::GetReqdRelationalProps(exprhdl.Prp());
	CColRefArray *pdrgpcrOutputStats = prprel->PcrsStat()->Pdrgpcr(mp);

	CColRef2dArray *pdrgpdrgpcrInputStats = GPOS_NEW(mp) CColRef2dArray(mp);
	/* 
	 * In CLogicalUnionAll, output columns are the same as the input columns
	 * of the first union all child
	 */
	pdrgpdrgpcrInputStats->Append(pdrgpcrOutputStats);
	/*
	 * We iteratively remap the stats columns for the 2nd (index 1) union all
	 * child and subsequent ones. Each of the stat columns, which are a subset
	 * of output columns, is mapped to the corresponding input column. The 
	 * first (index 0) union all child doesn't require remapping, because union
	 * all output columns align directly with the input columns of the first 
	 * union all child.
	 */
	const ULONG arity = exprhdl.Arity();
	for (ULONG ul = 1; ul < arity; ul++)
	{
		UlongToColRefMap *mapping =
			CUtils::PhmulcrMapping(mp, pdrgpcrOutput, (*pdrgpdrgpcrInput)[ul]);
		CColRefArray *pdrgpcrInputStats =
			CUtils::PdrgpcrRemap(mp, pdrgpcrOutputStats, mapping, true);
		pdrgpdrgpcrInputStats->Append(pdrgpcrInputStats);
		mapping->Release();
	}

	IStatistics *result = PstatsDeriveUnionAll(mp, exprhdl, pdrgpcrOutputStats,
											   pdrgpdrgpcrInputStats);
	pdrgpdrgpcrInputStats->Release();

	return result;
}

//---------------------------------------------------------------------------
//	@function:
//		CLogicalUnionAll::PcrsStat
//
//	@doc:
//		compute required stats columns of the n-th child
//
//---------------------------------------------------------------------------
CColRefSet *
CLogicalUnionAll::PcrsStat(CMemoryPool *mp,
						   CExpressionHandle &,	 // exprhdl
						   CColRefSet *pcrsInput, ULONG child_index) const
{
	/*
	 * We only need compute stats for columns required from parent of UnionAll
	 * operator. But the output columns of UnionAll are different from it`s 
	 * children, so we need to remap the stats columns to the corresponding input.
	 */
	if (0 == pcrsInput->Size())
	{
		pcrsInput->AddRef();
		return pcrsInput;
	}

	CColRefSet *pcrsChildInput = (*m_pdrgpcrsInput)[child_index];
	if (pcrsChildInput->ContainsAll(pcrsInput))
	{
		pcrsInput->AddRef();
		return pcrsInput;
	}
	else
	{
		UlongToColRefMap *mapping = CUtils::PhmulcrMapping(
			mp, m_pdrgpcrOutput, (*m_pdrgpdrgpcrInput)[child_index]);
		CColRefSet *pcrs = CUtils::PcrsRemap(mp, pcrsInput, mapping, true);
		mapping->Release();
		return pcrs;
	}
}

// EOF
