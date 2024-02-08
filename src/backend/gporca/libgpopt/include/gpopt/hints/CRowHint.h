//---------------------------------------------------------------------------
//	Greenplum Database
//	Copyright (c) 2023 VMware, Inc. or its affiliates. All Rights Reserved.
//
//	@filename:
//		CRowHint.h
//
//	@doc:
//		CRowHint represents an ORCA optimizer hint that allows the user to
//		effect join result cardinality estimates which in turn can effect the
//		plan shape. Syntax mirrors pg_hint_plan.
//
//	Example:
//		/*+ Rows(a b #10) */ SELECT... ; Sets rows of join result to 10
//		/*+ Rows(a b +10) */ SELECT... ; Increments row number by 10
//		/*+ Rows(a b -10) */ SELECT... ; Subtracts 10 from the row number.
//		/*+ Rows(a b *10) */ SELECT... ; Makes the number 10 times larger.
//
//	In DXL these are serialized into the below format:
//		<dxl:RowHint Alias="a,b" Rows="10.000000" Kind="Absolute"/>
//		<dxl:RowHint Alias="a,b" Rows="10.000000" Kind="Add"/>
//		<dxl:RowHint Alias="a,b" Rows="10.000000" Kind="Sub"/>
//		<dxl:RowHint Alias="a,b" Rows="10.000000" Kind="Multi"/>
//---------------------------------------------------------------------------
#ifndef GPOS_CRowHint_H
#define GPOS_CRowHint_H

#include "gpos/base.h"
#include "gpos/common/CDynamicPtrArray.h"
#include "gpos/common/CRefCount.h"

#include "gpopt/exception.h"
#include "gpopt/hints/IHint.h"
#include "gpopt/operators/COperator.h"
#include "naucrates/dxl/xml/CXMLSerializer.h"

namespace gpopt
{
class CRowHint : public IHint, public DbgPrintMixin<CRowHint>
{
public:
	enum RowsValueType
	{
		RVT_ABSOLUTE,
		RVT_ADD,
		RVT_SUB,
		RVT_MULTI,

		SENTINEL
	};

private:
	CMemoryPool *m_mp;

	// m_aliases contains sorted alias names.
	//
	// Row hint is order agnostic. Equivalent hints:
	//     Rows(a b #42)
	//     Rows(b a #42)
	StringPtrArray *m_aliases{nullptr};

	CDouble m_rows;

	RowsValueType m_type;

public:
	CRowHint(CMemoryPool *mp, StringPtrArray *aliases, CDouble rows,
			 RowsValueType type)
		: m_mp(mp), m_aliases(aliases), m_rows(rows), m_type(type)
	{
		aliases->Sort(CWStringBase::Compare);
	}

	~CRowHint() override
	{
		CRefCount::SafeRelease(m_aliases);
	}

	const StringPtrArray *
	GetAliasNames() const
	{
		return m_aliases;
	}

	RowsValueType
	GetType() const
	{
		return m_type;
	}

	//---------------------------------------------------------------------------
	//	@function:
	//		CRowHint::ComputeRows
	//
	//	@doc:
	//		Given an input value, evaluate the hint, then return the result.
	//
	//---------------------------------------------------------------------------
	CDouble
	ComputeRows(CDouble input) const
	{
		CDouble result(-1);
		switch (m_type)
		{
			case RVT_ABSOLUTE:
			{
				result = m_rows;
				break;
			}
			case RVT_ADD:
			{
				result = input + m_rows;
				break;
			}
			case RVT_SUB:
			{
				result = input - m_rows;
				break;
			}
			case RVT_MULTI:
			{
				result = input * m_rows;
				break;
			}
			default:
			{
				CWStringDynamic *error_message = GPOS_NEW(m_mp)
					CWStringDynamic(m_mp, GPOS_WSZ_LIT("Unknown row type"));

				GPOS_RAISE(gpopt::ExmaGPOPT, gpopt::ExmiUnsupportedOp,
						   error_message->GetBuffer());
			}
		}
		return result;
	}

	IOstream &OsPrint(IOstream &os) const;

	void Serialize(CXMLSerializer *xml_serializer) const;
};

using RowHintList = CDynamicPtrArray<CRowHint, CleanupRelease>;

}  // namespace gpopt

#endif	// !GPOS_CRowHint_H

// EOF
